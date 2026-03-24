## Objetivo del proyecto

Implementar un plugin de sincronizacion satelital para este fork de ChartDB
que permita sincronizar diagramas con una API remota, sin romper la compatibilidad
con el upstream (https://github.com/chartdb/chartdb).

## Stack tecnologico

- React + TypeScript
- Dexie.js (IndexedDB wrapper)
- Vite (variables de entorno con prefijo VITE\_)
- ahooks EventEmitter (ya usado en ChartDBContext)
- nanoid para IDs de entidades

## Regla de oro — Aislamiento del plugin

TODO el codigo nuevo va en: src/sync-plugin/
El UNICO archivo del core que se puede modificar es: src/pages/editor-page/editor-page.tsx
NUNCA modificar los archivos de versioning de Dexie (db.version(1)...db.version(9))
NUNCA modificar: storage-provider.tsx, chartdb-provider.tsx ni ningun contexto existente

## Estructura objetivo del plugin

src/sync-plugin/
api/
sync-api.ts <- cliente HTTP para la API remota
context/
sync-context/
sync-context.tsx
sync-provider.tsx
sync-storage-context/
sync-storage-provider.tsx <- ADAPTADOR CENTRAL
hooks/
use-sync.ts
types/
sync-types.ts

## Tablas Dexie a sincronizar

- diagrams
- db_tables
- db_relationships
- db_dependencies

## Conflictos: estrategia Last-Write-Wins

- diagrams -> comparar updatedAt (Date)
- tablas, fields, relaciones, dependencias -> comparar createdAt (number/timestamp)
- Para upserts usar putTable, updateRelationship, updateDependency ya existentes
- Para cargar estado remoto en UI sin recargar: usar loadDiagramFromData

## Variables de entorno a anadir en src/lib/env.ts

- VITE_SYNC_API_URL
- VITE_SYNC_ENABLED (boolean, activa/desactiva el plugin)

## Cambio en editor-page.tsx

Reemplazar <StorageProvider> por <SyncStorageProvider> de forma condicional
segun VITE_SYNC_ENABLED. SyncStorageProvider usa StorageProvider internamente.

## Interfaz que debe implementar SyncStorageProvider

Debe exponer exactamente las mismas funciones que StorageContext:
getConfig, updateConfig, addDiagram, listDiagrams, getDiagram, updateDiagram,
deleteDiagram, addTable, getTable, updateTable, putTable, deleteTable, listTables,
addRelationship, getRelationship, updateRelationship, deleteRelationship,
listRelationships, deleteDiagramTables, deleteDiagramRelationships,
addDependency, getDependency, updateDependency, deleteDependency,
listDependencies, deleteDiagramDependencies

## Sistema de eventos existente (NO modificar, solo suscribirse)

Eventos disponibles en ChartDBContext: add_tables, update_table, remove_tables,
add_field, remove_field, load_diagram

## API de sincronizacion (Go + Fiber + GORM)

El backend vive en la carpeta sync-api/ en la raiz del repositorio.
Endpoints principales:

- POST /api/sync/push -> envia cambios locales al servidor (Last-Write-Wins)
- GET /api/sync/pull/:diagramId -> trae estado remoto completo al arrancar
  Autenticacion: header X-API-Secret con valor de variable de entorno API_SECRET

## Estado de las tareas

Consulta siempre TASK_LIST.md en la raiz antes de empezar.
Tras cada ejecucion, actualiza ese archivo: marca completadas con [x],
anade nuevas subtareas si aparecen, y haz commit del archivo actualizado.
