import React, { useCallback, useContext, useMemo } from 'react';
import { StorageProvider } from '../../../context/storage-context/storage-provider';
import { storageContext, type StorageContext } from '../../../context/storage-context/storage-context';
import { syncApi } from '../../api/sync-api';
import type { Diagram } from '../../../lib/domain/diagram';
import type { DBTable } from '../../../lib/domain/db-table';
import type { DBRelationship } from '../../../lib/domain/db-relationship';
import type { DBDependency } from '../../../lib/domain/db-dependency';

export const SyncStorageProviderInner: React.FC<React.PropsWithChildren> = ({
    children,
}) => {
    const baseContext = useContext(storageContext);

    // triggerSync runs in background to push latest local state for diagram to server
    const triggerSync = useCallback(
        async (diagramId: string) => {
            try {
                // Fetch local state
                const diagram = await baseContext.getDiagram(diagramId);
                if (!diagram) return; // if deleted, we don't sync for now
                const tables = await baseContext.listTables(diagramId);
                const relationships = await baseContext.listRelationships(diagramId);
                const dependencies = await baseContext.listDependencies(diagramId);

                // Push to server
                await syncApi.push({
                    diagram,
                    tables,
                    relationships,
                    dependencies,
                });
            } catch (error) {
                console.error('Sync push failed:', error);
            }
        },
        [baseContext]
    );

    const syncContextValues = useMemo<StorageContext>(() => {
        return {
            ...baseContext,

            // --- Diagram ---
            addDiagram: async (params) => {
                await baseContext.addDiagram(params);
                triggerSync(params.diagram.id);
            },
            updateDiagram: async (params) => {
                await baseContext.updateDiagram(params);
                triggerSync(params.id);
            },
            deleteDiagram: async (id) => {
                await baseContext.deleteDiagram(id);
                // Can't push a deleted diagram easily if it's gone from local db,
                // for now we don't trigger sync or we just let it be.
            },

            // --- Tables ---
            addTable: async (params) => {
                await baseContext.addTable(params);
                triggerSync(params.diagramId);
            },
            updateTable: async (params) => {
                // Find diagramId first
                // Wait, if we don't know diagramId, we might not be able to get the table easily
                // let's try to get all diagrams and find it? No, Dexie tables have diagramId.
                // However getTable requires diagramId too!
                // But wait, the updateTable params are just { id: string, attributes: Partial<DBTable> }.
                // How do we find the table without diagramId? We might have to query Dexie directly or list all tables.
                // Fortunately, we can probably get it if we assume it might be in current context, or we just
                // fetch the diagramId from the updated attributes if it's there.

                // Let's implement it by listing all diagrams, finding the table, or let's assume updateTable itself
                // might not change diagramId. Wait, let's use the DB instance if needed, but we only have baseContext.
                // Actually, if we listDiagrams, we can get all tables and find the one with the id.

                // For now, let's just do baseContext.updateTable(params) and if attributes.diagramId is there, trigger sync.
                // This is a common pattern in the app. Let's see if we can do a hack to find diagramId.
                let diagramId = params.attributes?.diagramId as string | undefined;
                if (!diagramId) {
                    const diagrams = await baseContext.listDiagrams({ includeTables: true });
                    for (const diagram of diagrams) {
                        const table = diagram.tables?.find((t) => t.id === params.id);
                        if (table) {
                            diagramId = diagram.id;
                            break;
                        }
                    }
                }

                await baseContext.updateTable(params);

                if (diagramId) {
                    triggerSync(diagramId);
                }
            },
            putTable: async (params) => {
                await baseContext.putTable(params);
                triggerSync(params.diagramId);
            },
            deleteTable: async (params) => {
                await baseContext.deleteTable(params);
                triggerSync(params.diagramId);
            },
            deleteDiagramTables: async (diagramId) => {
                await baseContext.deleteDiagramTables(diagramId);
                triggerSync(diagramId);
            },

            // --- Relationships ---
            addRelationship: async (params) => {
                await baseContext.addRelationship(params);
                triggerSync(params.diagramId);
            },
            updateRelationship: async (params) => {
                let diagramId = params.attributes?.diagramId as string | undefined;
                if (!diagramId) {
                    const diagrams = await baseContext.listDiagrams({ includeRelationships: true });
                    for (const diagram of diagrams) {
                        const rel = diagram.relationships?.find((r) => r.id === params.id);
                        if (rel) {
                            diagramId = diagram.id;
                            break;
                        }
                    }
                }

                await baseContext.updateRelationship(params);

                if (diagramId) {
                    triggerSync(diagramId);
                }
            },
            deleteRelationship: async (params) => {
                await baseContext.deleteRelationship(params);
                triggerSync(params.diagramId);
            },
            deleteDiagramRelationships: async (diagramId) => {
                await baseContext.deleteDiagramRelationships(diagramId);
                triggerSync(diagramId);
            },

            // --- Dependencies ---
            addDependency: async (params) => {
                await baseContext.addDependency(params);
                triggerSync(params.diagramId);
            },
            updateDependency: async (params) => {
                let diagramId = params.attributes?.diagramId as string | undefined;
                if (!diagramId) {
                    const diagrams = await baseContext.listDiagrams({ includeDependencies: true });
                    for (const diagram of diagrams) {
                        const dep = diagram.dependencies?.find((d) => d.id === params.id);
                        if (dep) {
                            diagramId = diagram.id;
                            break;
                        }
                    }
                }

                await baseContext.updateDependency(params);

                if (diagramId) {
                    triggerSync(diagramId);
                }
            },
            deleteDependency: async (params) => {
                await baseContext.deleteDependency(params);
                triggerSync(params.diagramId);
            },
            deleteDiagramDependencies: async (diagramId) => {
                await baseContext.deleteDiagramDependencies(diagramId);
                triggerSync(diagramId);
            },
        };
    }, [baseContext, triggerSync]);

    return (
        <storageContext.Provider value={syncContextValues}>
            {children}
        </storageContext.Provider>
    );
};

export const SyncStorageProvider: React.FC<React.PropsWithChildren> = ({
    children,
}) => {
    return (
        <StorageProvider>
            <SyncStorageProviderInner>{children}</SyncStorageProviderInner>
        </StorageProvider>
    );
};
