# Guía para crear un Plugin de Sincronización Satelital para ChartDB

Después de analizar en profundidad la arquitectura del repositorio, aquí está el plan completo para crear el adaptador de sincronización sin romper la compatibilidad con el upstream.

---

## 1. Entender la Arquitectura de Almacenamiento

El punto central de todo el sistema de persistencia es el `StorageProvider`. Toda la aplicación usa **Dexie.js** (wrapper de IndexedDB) y expone sus operaciones a través de un React Context. [1](#0-0)

Las tablas en Dexie que necesitas sincronizar son exactamente 4:

- `diagrams`
- `db_tables`
- `db_relationships`
- `db_dependencies` [2](#0-1)

El `StorageContext` expone una interfaz limpia con todas las operaciones CRUD (addDiagram, updateTable, deleteRelationship, etc.) que **ya está desacoplada de Dexie**. Esto es la clave para hacer el plugin sin tocar el núcleo. [3](#0-2)

---

## 2. Cómo Está Conectado al Estado de la App

El `ChartDBProvider` consume el `StorageContext` via `useStorage()` para todas sus operaciones de escritura. Por ejemplo, al agregar una tabla: [4](#0-3)

Al actualizar un campo: [5](#0-4)

Esto significa que **si interceptas el StorageContext, interceptas TODA la persistencia** sin tocar la lógica de negocio.

---

## 3. El Sistema de Eventos (tu mejor aliado)

ChartDB ya tiene un sistema de eventos con `EventEmitter` de `ahooks` que emite cambios en tiempo real: [6](#0-5)

Estos eventos se emiten dentro del provider en operaciones como `addTables`, `removeTables`, `updateTable`, `addField`, `removeField` y `loadDiagram`: [7](#0-6) [8](#0-7)

---

## 4. Estructura del Plugin (Sin Colisiones con Upstream)

La estrategia para no tener colisiones con el upstream en el fork es **aislar todo el código de sync en un directorio separado** y **modificar un único archivo del core**: el `editor-page.tsx`.

```
src/
  sync-plugin/               ← TODO TU CÓDIGO NUEVO AQUÍ
    api/
      sync-api.ts
    context/
      sync-context/
        sync-context.tsx
        sync-provider.tsx
      sync-storage-context/
        sync-storage-provider.tsx  ← EL ADAPTADOR CENTRAL
    hooks/
      use-sync.ts
    types/
      sync-types.ts
```

El único archivo del core que debes modificar es el que orquesta todos los providers: [9](#0-8)

Aquí solo cambias `<StorageProvider>` por `<SyncStorageProvider>` de forma condicional según una variable de entorno.

---

## 5. Variables de Entorno (Patrón ya usado en el proyecto)

El proyecto ya usa este patrón para features opcionales en `src/lib/env.ts`: [10](#0-9)

Agrega al final: `VITE_SYNC_API_URL` y `VITE_SYNC_ENABLED`. Así puedes activar/desactivar el plugin sin modificar la lógica.

---

## 6. El Adaptador Central: `SyncStorageProvider`

### Cómo Funciona

El `SyncStorageProvider` debe **implementar la misma interfaz del StorageContext** pero interceptar cada operación de escritura para también enviar los cambios a la API. La interfaz completa a implementar son las funciones ya existentes: [11](#0-10) [12](#0-11) [13](#0-12) [14](#0-13)

Para el **pull/merge al inicio**, el método clave es `putTable`, que ya existe como un upsert en Dexie: [15](#0-14)

Usa `putTable`, `updateRelationship`, `updateDependency` para hacer upserts cuando lleguen datos remotos, evitando duplicados.

---

## 7. Estrategia para Evitar Colisiones (Conflict Resolution)

### Datos que ya tienes disponibles

Todos los modelos ya tienen `updatedAt` en los diagramas: [16](#0-15)

Y `createdAt` en tablas, campos, relaciones y dependencias: [17](#0-16) [18](#0-17) [19](#0-18) [20](#0-19)

### Estrategia Recomendada: Last-Write-Wins por Entidad

```
diagrama.updatedAt   → Comparar local vs remoto para decidir quién gana
tabla.createdAt      → ID + timestamp como key de deduplicación
```

El flujo es:

1. Al arrancar, pull del API remoto
2. Comparar `diagram.updatedAt` local vs remoto
3. Si el remoto es más reciente → usar `putTable`/`addRelationship` del storage para hacer upsert entidad por entidad
4. Si el local es más reciente → push al API
5. En caso de conflicto simultáneo → el servidor decide (envías `updatedAt` en el header y el servidor rechaza si hay una versión más nueva)

Para **cargar el estado remoto en la UI sin recargar la página**, usa `loadDiagramFromData` que ya existe: [21](#0-20)

---

## 8. Diagrama de Arquitectura Completo

```mermaid
graph TD
    "editor-page.tsx" --> "SyncStorageProvider (TUYO)"
    "SyncStorageProvider (TUYO)" --> "Dexie IndexedDB (original)"
    "SyncStorageProvider (TUYO)" --> "SyncAPI (TUYO)"
    "SyncAPI (TUYO)" --> "Tu API Remota"
    "StorageContext Interface" --> "SyncStorageProvider (TUYO)"
    "ChartDBProvider (original)" --> "StorageContext Interface"
    "SyncProvider (TUYO)" --> "ChartDB events (EventEmitter)"
    "SyncProvider (TUYO)" --> "loadDiagramFromData"
    "loadDiagramFromData" --> "ChartDBProvider (original)"
```

---

## 9. Estrategia de Merge con Upstream sin Colisiones

```
tu-fork/
  main          ← tu rama productiva (sync activado)
  upstream-sync ← rama que trackea cambios de upstream
```

```bash
# En tu fork:
git remote add upstream https://github.com/chartdb/chartdb.git
git fetch upstream
git merge upstream/main
```

**El único conflicto posible** estará en `editor-page.tsx` (donde agregas `SyncStorageProvider`), ya que todos tus archivos nuevos están en `src/sync-plugin/` (directorio que upstream nunca toca). Este conflicto es trivial de resolver: solo debes asegurarte de que `SyncStorageProvider` sigue envolviendo a `StorageProvider` después del merge.

La composición de providers que muestra el archivo ya sigue un patrón de capas: [9](#0-8)

Tu único cambio es reemplazar `<StorageProvider>` por `<SyncStorageProvider>` en esa composición, que también puede recibir el `StorageProvider` original internamente para reutilizarlo.

---

## Notes

- **Dexie `putTable`** es tu mejor aliado para el merge: hace un INSERT-OR-UPDATE atómico, lo que elimina la necesidad de verificar si el registro ya existe antes de sincronizar.
- Los **IDs** de todas las entidades son `nanoid` generados en el cliente. [22](#0-21) Esto es seguro para sync distribuido porque la probabilidad de colisión de nanoid es extremadamente baja.
- El sistema de **Diff** ya existente (`DiffContext`) podría reutilizarse para mostrar visualmente al usuario qué cambios llegaron desde otros dispositivos. [23](#0-22)
- **No modifiques** los archivos de versioning de Dexie (`db.version(1)...db.version(9)`). Si agregas campos de sync (como `syncedAt`), hazlo solo en tu `SyncStorageProvider` en tablas separadas de Dexie o en `localStorage`. [24](#0-23)

### Citations

**File:** src/context/storage-context/storage-provider.tsx (L12-36)

```typescript
export const StorageProvider: React.FC<React.PropsWithChildren> = ({
    children,
}) => {
    const db = new Dexie('ChartDB') as Dexie & {
        diagrams: EntityTable<
            Diagram,
            'id' // primary key "id" (for the typings only)
        >;
        db_tables: EntityTable<
            DBTable & { diagramId: string },
            'id' // primary key "id" (for the typings only)
        >;
        db_relationships: EntityTable<
            DBRelationship & { diagramId: string },
            'id' // primary key "id" (for the typings only)
        >;
        db_dependencies: EntityTable<
            DBDependency & { diagramId: string },
            'id' // primary key "id" (for the typings only)
        >;
        config: EntityTable<
            ChartDBConfig & { id: number },
            'id' // primary key "id" (for the typings only)
        >;
    };
```

**File:** src/context/storage-context/storage-provider.tsx (L39-135)

```typescript
db.version(1).stores({
  diagrams: "++id, name, databaseType, createdAt, updatedAt",
  db_tables:
    "++id, diagramId, name, x, y, fields, indexes, color, createdAt, width",
  db_relationships:
    "++id, diagramId, name, sourceTableId, targetTableId, sourceFieldId, targetFieldId, type, createdAt",
  config: "++id, defaultDiagramId",
});

db.version(2).upgrade((tx) =>
  tx
    .table<DBTable & { diagramId: string }>("db_tables")
    .toCollection()
    .modify((table) => {
      for (const field of table.fields) {
        field.type = {
          // @ts-expect-error string before
          id: (field.type as string).split(" ").join("_"),
          // @ts-expect-error string before
          name: field.type,
        };
      }
    }),
);

db.version(3).stores({
  diagrams: "++id, name, databaseType, databaseEdition, createdAt, updatedAt",
  db_tables:
    "++id, diagramId, name, x, y, fields, indexes, color, createdAt, width",
  db_relationships:
    "++id, diagramId, name, sourceTableId, targetTableId, sourceFieldId, targetFieldId, type, createdAt",
  config: "++id, defaultDiagramId",
});

db.version(4).stores({
  diagrams: "++id, name, databaseType, databaseEdition, createdAt, updatedAt",
  db_tables:
    "++id, diagramId, name, x, y, fields, indexes, color, createdAt, width, comment",
  db_relationships:
    "++id, diagramId, name, sourceTableId, targetTableId, sourceFieldId, targetFieldId, type, createdAt",
  config: "++id, defaultDiagramId",
});

db.version(5).stores({
  diagrams: "++id, name, databaseType, databaseEdition, createdAt, updatedAt",
  db_tables:
    "++id, diagramId, name, schema, x, y, fields, indexes, color, createdAt, width, comment",
  db_relationships:
    "++id, diagramId, name, sourceSchema, sourceTableId, targetSchema, targetTableId, sourceFieldId, targetFieldId, type, createdAt",
  config: "++id, defaultDiagramId",
});

db.version(6).upgrade((tx) =>
  tx
    .table<DBRelationship & { diagramId: string }>("db_relationships")
    .toCollection()
    .modify((relationship, ref) => {
      const {
        sourceCardinality,
        targetCardinality,
      } = // @ts-expect-error string before
        determineCardinalities(relationship.type ?? "one_to_one");

      relationship.sourceCardinality = sourceCardinality;
      relationship.targetCardinality = targetCardinality;

      // @ts-expect-error string before
      delete ref.value.type;
    }),
);

db.version(7).stores({
  diagrams: "++id, name, databaseType, databaseEdition, createdAt, updatedAt",
  db_tables:
    "++id, diagramId, name, schema, x, y, fields, indexes, color, createdAt, width, comment",
  db_relationships:
    "++id, diagramId, name, sourceSchema, sourceTableId, targetSchema, targetTableId, sourceFieldId, targetFieldId, type, createdAt",
  db_dependencies:
    "++id, diagramId, schema, tableId, dependentSchema, dependentTableId, createdAt",
  config: "++id, defaultDiagramId",
});

db.version(8).stores({
  diagrams: "++id, name, databaseType, databaseEdition, createdAt, updatedAt",
  db_tables:
    "++id, diagramId, name, schema, x, y, fields, indexes, color, createdAt, width, comment, isView, isMaterializedView, order",
  db_relationships:
    "++id, diagramId, name, sourceSchema, sourceTableId, targetSchema, targetTableId, sourceFieldId, targetFieldId, type, createdAt",
  db_dependencies:
    "++id, diagramId, schema, tableId, dependentSchema, dependentTableId, createdAt",
  config: "++id, defaultDiagramId",
});
```

**File:** src/context/storage-context/storage-provider.tsx (L164-213)

```typescript
const getConfig: StorageContext["getConfig"] = async (): Promise<
  ChartDBConfig | undefined
> => {
  return await db.config.get(1);
};

const updateConfig: StorageContext["updateConfig"] = async (
  config: Partial<ChartDBConfig>,
) => {
  await db.config.update(1, config);
};

const addDiagram: StorageContext["addDiagram"] = async ({
  diagram,
}: {
  diagram: Diagram;
}) => {
  const promises = [];
  promises.push(
    db.diagrams.add({
      id: diagram.id,
      name: diagram.name,
      databaseType: diagram.databaseType,
      databaseEdition: diagram.databaseEdition,
      createdAt: diagram.createdAt,
      updatedAt: diagram.updatedAt,
    }),
  );

  const tables = diagram.tables ?? [];
  promises.push(
    ...tables.map((table) => addTable({ diagramId: diagram.id, table })),
  );

  const relationships = diagram.relationships ?? [];
  promises.push(
    ...relationships.map((relationship) =>
      addRelationship({ diagramId: diagram.id, relationship }),
    ),
  );

  const dependencies = diagram.dependencies ?? [];
  promises.push(
    ...dependencies.map((dependency) =>
      addDependency({ diagramId: diagram.id, dependency }),
    ),
  );

  await Promise.all(promises);
};
```

**File:** src/context/storage-context/storage-provider.tsx (L291-340)

```typescript
const updateDiagram: StorageContext["updateDiagram"] = async ({
  id,
  attributes,
}: {
  id: string;
  attributes: Partial<Diagram>;
}) => {
  await db.diagrams.update(id, attributes);

  if (attributes.id) {
    await Promise.all([
      db.db_tables
        .where("diagramId")
        .equals(id)
        .modify({ diagramId: attributes.id }),
      db.db_relationships
        .where("diagramId")
        .equals(id)
        .modify({ diagramId: attributes.id }),
      db.db_dependencies
        .where("diagramId")
        .equals(id)
        .modify({ diagramId: attributes.id }),
    ]);
  }
};

const deleteDiagram: StorageContext["deleteDiagram"] = async (id: string) => {
  await Promise.all([
    db.diagrams.delete(id),
    db.db_tables.where("diagramId").equals(id).delete(),
    db.db_relationships.where("diagramId").equals(id).delete(),
    db.db_dependencies.where("diagramId").equals(id).delete(),
  ]);
};

const addTable: StorageContext["addTable"] = async ({
  diagramId,
  table,
}: {
  diagramId: string;
  table: DBTable;
}) => {
  await db.db_tables.add({
    ...table,
    diagramId,
  });
};
```

**File:** src/context/storage-context/storage-provider.tsx (L365-370)

```typescript
const putTable: StorageContext["putTable"] = async ({ diagramId, table }) => {
  await db.db_tables.put({ ...table, diagramId });
};
```

**File:** src/context/storage-context/storage-provider.tsx (L394-413)

```typescript
const addRelationship: StorageContext["addRelationship"] = async ({
  diagramId,
  relationship,
}: {
  diagramId: string;
  relationship: DBRelationship;
}) => {
  await db.db_relationships.add({
    ...relationship,
    diagramId,
  });
};

const deleteDiagramRelationships: StorageContext["deleteDiagramRelationships"] =
  async (diagramId: string) => {
    await db.db_relationships.where("diagramId").equals(diagramId).delete();
  };
```

**File:** src/context/storage-context/storage-provider.tsx (L459-497)

```typescript
const addDependency: StorageContext["addDependency"] = async ({
  diagramId,
  dependency,
}) => {
  await db.db_dependencies.add({
    ...dependency,
    diagramId,
  });
};

const getDependency: StorageContext["getDependency"] = async ({
  diagramId,
  id,
}) => {
  return await db.db_dependencies.get({ id, diagramId });
};

const updateDependency: StorageContext["updateDependency"] = async ({
  id,
  attributes,
}) => {
  await db.db_dependencies.update(id, attributes);
};

const deleteDependency: StorageContext["deleteDependency"] = async ({
  diagramId,
  id,
}) => {
  await db.db_dependencies.where({ id, diagramId }).delete();
};

const listDependencies: StorageContext["listDependencies"] = async (
  diagramId,
) => {
  return await db.db_dependencies
    .where("diagramId")
    .equals(diagramId)
    .toArray();
};
```

**File:** src/context/storage-context/storage-provider.tsx (L507-540)

```typescript
    return (
        <storageContext.Provider
            value={{
                getConfig,
                updateConfig,
                addDiagram,
                listDiagrams,
                getDiagram,
                updateDiagram,
                deleteDiagram,
                addTable,
                getTable,
                updateTable,
                putTable,
                deleteTable,
                listTables,
                addRelationship,
                getRelationship,
                updateRelationship,
                deleteRelationship,
                listRelationships,
                deleteDiagramTables,
                deleteDiagramRelationships,
                addDependency,
                getDependency,
                updateDependency,
                deleteDependency,
                listDependencies,
                deleteDiagramDependencies,
            }}
        >
            {children}
        </storageContext.Provider>
    );
```

**File:** src/context/chartdb-context/chartdb-provider.tsx (L285-307)

```typescript
const addTables: ChartDBContext["addTables"] = useCallback(
  async (tables: DBTable[], options = { updateHistory: true }) => {
    setTables((currentTables) => [...currentTables, ...tables]);
    const updatedAt = new Date();
    setDiagramUpdatedAt(updatedAt);
    await Promise.all([
      db.updateDiagram({ id: diagramId, attributes: { updatedAt } }),
      ...tables.map((table) => db.addTable({ diagramId, table })),
    ]);

    events.emit({ action: "add_tables", data: { tables } });

    if (options.updateHistory) {
      addUndoAction({
        action: "addTables",
        redoData: { tables },
        undoData: { tableIds: tables.map((t) => t.id) },
      });
      resetRedoStack();
    }
  },
  [db, diagramId, setTables, addUndoAction, resetRedoStack, events],
);
```

**File:** src/context/chartdb-context/chartdb-provider.tsx (L316-348)

```typescript
    const createTable: ChartDBContext['createTable'] = useCallback(
        async (attributes) => {
            const table: DBTable = {
                id: generateId(),
                name: `table_${tables.length + 1}`,
                x: 0,
                y: 0,
                fields: [
                    {
                        id: generateId(),
                        name: 'id',
                        type:
                            databaseType === DatabaseType.SQLITE
                                ? { id: 'integer', name: 'integer' }
                                : { id: 'bigint', name: 'bigint' },
                        unique: true,
                        nullable: false,
                        primaryKey: true,
                        createdAt: Date.now(),
                    },
                ],
                indexes: [],
                color: randomColor(),
                createdAt: Date.now(),
                isView: false,
                order: tables.length,
                ...attributes,
            };
            await addTable(table);

            return table;
        },
        [addTable, tables, databaseType]
```

**File:** src/context/chartdb-context/chartdb-provider.tsx (L627-681)

```typescript
    const updateField: ChartDBContext['updateField'] = useCallback(
        async (
            tableId: string,
            fieldId: string,
            field: Partial<DBField>,
            options = { updateHistory: true }
        ) => {
            const prevField = getField(tableId, fieldId);
            setTables((tables) =>
                tables.map((table) =>
                    table.id === tableId
                        ? {
                              ...table,
                              fields: table.fields.map((f) =>
                                  f.id === fieldId ? { ...f, ...field } : f
                              ),
                          }
                        : table
                )
            );

            const table = await db.getTable({ diagramId, id: tableId });
            if (!table) {
                return;
            }

            const updatedAt = new Date();
            setDiagramUpdatedAt(updatedAt);
            await Promise.all([
                db.updateDiagram({ id: diagramId, attributes: { updatedAt } }),
                db.updateTable({
                    id: tableId,
                    attributes: {
                        ...table,
                        fields: table.fields.map((f) =>
                            f.id === fieldId ? { ...f, ...field } : f
                        ),
                    },
                }),
            ]);

            if (!!prevField && options.updateHistory) {
                addUndoAction({
                    action: 'updateField',
                    redoData: {
                        tableId,
                        fieldId,
                        field: { ...prevField, ...field },
                    },
                    undoData: { tableId, fieldId, field: prevField },
                });
                resetRedoStack();
            }
        },
        [db, diagramId, setTables, addUndoAction, resetRedoStack, getField]
```

**File:** src/context/chartdb-context/chartdb-provider.tsx (L1366-1410)

```typescript
const loadDiagramFromData: ChartDBContext["loadDiagramFromData"] = useCallback(
  async (diagram) => {
    setDiagramId(diagram.id);
    setDiagramName(diagram.name);
    setDatabaseType(diagram.databaseType);
    setDatabaseEdition(diagram.databaseEdition);
    setTables(diagram?.tables ?? []);
    setRelationships(diagram?.relationships ?? []);
    setDependencies(diagram?.dependencies ?? []);
    setDiagramCreatedAt(diagram.createdAt);
    setDiagramUpdatedAt(diagram.updatedAt);

    events.emit({ action: "load_diagram", data: { diagram } });
  },
  [
    setDiagramId,
    setDiagramName,
    setDatabaseType,
    setDatabaseEdition,
    setTables,
    setRelationships,
    setDependencies,
    setDiagramCreatedAt,
    setDiagramUpdatedAt,
    events,
  ],
);

const loadDiagram: ChartDBContext["loadDiagram"] = useCallback(
  async (diagramId: string) => {
    const diagram = await db.getDiagram(diagramId, {
      includeRelationships: true,
      includeTables: true,
      includeDependencies: true,
    });

    if (diagram) {
      loadDiagramFromData(diagram);
    }

    return diagram;
  },
  [db, loadDiagramFromData],
);
```

**File:** src/context/chartdb-context/chartdb-context.tsx (L14-63)

```typescript
export type ChartDBEventType =
  | "add_tables"
  | "update_table"
  | "remove_tables"
  | "add_field"
  | "remove_field"
  | "load_diagram";

export type ChartDBEventBase<T extends ChartDBEventType, D> = {
  action: T;
  data: D;
};

export type CreateTableEvent = ChartDBEventBase<
  "add_tables",
  { tables: DBTable[] }
>;

export type UpdateTableEvent = ChartDBEventBase<
  "update_table",
  { id: string; table: Partial<DBTable> }
>;

export type RemoveTableEvent = ChartDBEventBase<
  "remove_tables",
  { tableIds: string[] }
>;

export type AddFieldEvent = ChartDBEventBase<
  "add_field",
  { tableId: string; field: DBField; fields: DBField[] }
>;

export type RemoveFieldEvent = ChartDBEventBase<
  "remove_field",
  { tableId: string; fieldId: string; fields: DBField[] }
>;

export type LoadDiagramEvent = ChartDBEventBase<
  "load_diagram",
  { diagram: Diagram }
>;

export type ChartDBEvent =
  | CreateTableEvent
  | UpdateTableEvent
  | RemoveTableEvent
  | AddFieldEvent
  | RemoveFieldEvent
  | LoadDiagramEvent;
```

**File:** src/pages/editor-page/editor-page.tsx (L197-231)

```typescript
export const EditorPage: React.FC = () => (
    <LocalConfigProvider>
        <ThemeProvider>
            <FullScreenLoaderProvider>
                <LayoutProvider>
                    <StorageProvider>
                        <ConfigProvider>
                            <RedoUndoStackProvider>
                                <DiffProvider>
                                    <ChartDBProvider>
                                        <HistoryProvider>
                                            <ReactFlowProvider>
                                                <CanvasProvider>
                                                    <ExportImageProvider>
                                                        <AlertProvider>
                                                            <DialogProvider>
                                                                <KeyboardShortcutsProvider>
                                                                    <EditorPageComponent />
                                                                </KeyboardShortcutsProvider>
                                                            </DialogProvider>
                                                        </AlertProvider>
                                                    </ExportImageProvider>
                                                </CanvasProvider>
                                            </ReactFlowProvider>
                                        </HistoryProvider>
                                    </ChartDBProvider>
                                </DiffProvider>
                            </RedoUndoStackProvider>
                        </ConfigProvider>
                    </StorageProvider>
                </LayoutProvider>
            </FullScreenLoaderProvider>
        </ThemeProvider>
    </LocalConfigProvider>
);
```

**File:** src/lib/env.ts (L1-11)

```typescript
export const OPENAI_API_KEY: string = import.meta.env.VITE_OPENAI_API_KEY;
export const OPENAI_API_ENDPOINT: string = import.meta.env
  .VITE_OPENAI_API_ENDPOINT;
export const LLM_MODEL_NAME: string = import.meta.env.VITE_LLM_MODEL_NAME;
export const IS_CHARTDB_IO: boolean =
  import.meta.env.VITE_IS_CHARTDB_IO === "true";
export const APP_URL: string = import.meta.env.VITE_APP_URL;
export const HOST_URL: string = import.meta.env.VITE_HOST_URL ?? "";
export const HIDE_BUCKLE_DOT_DEV: boolean =
  (window?.env?.HIDE_BUCKLE_DOT_DEV ??
    import.meta.env.VITE_HIDE_BUCKLE_DOT_DEV) === "true";
```

**File:** src/lib/domain/db-table.ts (L29-44)

```typescript
export interface DBTable {
  id: string;
  name: string;
  schema?: string;
  x: number;
  y: number;
  fields: DBField[];
  indexes: DBIndex[];
  color: string;
  isView: boolean;
  isMaterializedView?: boolean;
  createdAt: number;
  width?: number;
  comments?: string;
  order?: number;
}
```

**File:** src/lib/domain/db-field.ts (L10-24)

```typescript
export interface DBField {
  id: string;
  name: string;
  type: DataType;
  primaryKey: boolean;
  unique: boolean;
  nullable: boolean;
  createdAt: number;
  characterMaximumLength?: string;
  precision?: number;
  scale?: number;
  default?: string;
  collation?: string;
  comments?: string;
}
```

**File:** src/lib/domain/db-relationship.ts (L11-23)

```typescript
export interface DBRelationship {
  id: string;
  name: string;
  sourceSchema?: string;
  sourceTableId: string;
  targetSchema?: string;
  targetTableId: string;
  sourceFieldId: string;
  targetFieldId: string;
  sourceCardinality: Cardinality;
  targetCardinality: Cardinality;
  createdAt: number;
}
```

**File:** src/lib/domain/db-dependency.ts (L12-19)

```typescript
export interface DBDependency {
  id: string;
  schema?: string;
  tableId: string;
  dependentSchema?: string;
  dependentTableId: string;
  createdAt: number;
}
```

**File:** src/context/diff-context/diff-context.tsx (L30-77)

```typescript
export interface DiffContext {
  newDiagram: Diagram | null;
  originalDiagram: Diagram | null;
  diffMap: DiffMap;
  hasDiff: boolean;

  calculateDiff: ({
    diagram,
    newDiagram,
  }: {
    diagram: Diagram;
    newDiagram: Diagram;
  }) => void;

  // table diff
  checkIfTableHasChange: ({ tableId }: { tableId: string }) => boolean;
  checkIfNewTable: ({ tableId }: { tableId: string }) => boolean;
  checkIfTableRemoved: ({ tableId }: { tableId: string }) => boolean;
  getTableNewName: ({ tableId }: { tableId: string }) => string | null;
  getTableNewColor: ({ tableId }: { tableId: string }) => string | null;

  // field diff
  checkIfFieldHasChange: ({
    tableId,
    fieldId,
  }: {
    tableId: string;
    fieldId: string;
  }) => boolean;
  checkIfFieldRemoved: ({ fieldId }: { fieldId: string }) => boolean;
  checkIfNewField: ({ fieldId }: { fieldId: string }) => boolean;
  getFieldNewName: ({ fieldId }: { fieldId: string }) => string | null;
  getFieldNewType: ({ fieldId }: { fieldId: string }) => DataType | null;

  // relationship diff
  checkIfNewRelationship: ({
    relationshipId,
  }: {
    relationshipId: string;
  }) => boolean;
  checkIfRelationshipRemoved: ({
    relationshipId,
  }: {
    relationshipId: string;
  }) => boolean;

  events: EventEmitter<DiffEvent>;
}
```
