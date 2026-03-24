import React, { useContext, useState, useEffect, useCallback, useRef } from 'react';
import { syncContext } from './sync-context';
import { chartDBContext, type LoadDiagramEvent } from '../../../context/chartdb-context/chartdb-context';
import { storageContext } from '../../../context/storage-context/storage-context';
import { syncApi } from '../../api/sync-api';

export const SyncProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
    const { events, loadDiagram } = useContext(chartDBContext);
    const store = useContext(storageContext);
    const [isSyncing, setIsSyncing] = useState(false);
    const isSyncingRef = useRef(false);

    const handleLoadDiagram = useCallback(async (event: LoadDiagramEvent) => {
        if (event.action !== 'load_diagram') return;

        const diagramId = event.data.diagram.id;

        if (isSyncingRef.current) return;
        isSyncingRef.current = true;
        setIsSyncing(true);

        // Fetch remote data
        try {
            const remoteData = await syncApi.pull(diagramId);

            if (!remoteData || !remoteData.diagram) {
                return;
            }

            const localDiagram = await store.getDiagram(diagramId);

            // Check conflict for diagram
            let isRemoteNewer = false;
            let currentLocalUpdatedAt = localDiagram ? localDiagram.updatedAt : new Date(0);

            // Check if remote data is newer
            const remoteDate = new Date(remoteData.diagram.updatedAt);
            const localDate = new Date(currentLocalUpdatedAt);

            if (!localDiagram || remoteDate > localDate) {
                isRemoteNewer = true;
            }

            if (isRemoteNewer) {
                // Upsert diagram
                if (!localDiagram) {
                    await store.addDiagram({ diagram: remoteData.diagram });
                } else {
                    await store.updateDiagram({ id: diagramId, attributes: remoteData.diagram });
                }

                // Compare tables
                const localTables = await store.listTables(diagramId);
                for (const remoteTable of remoteData.tables) {
                    const localTable = localTables.find(t => t.id === remoteTable.id);
                    if (!localTable || remoteTable.createdAt > localTable.createdAt) {
                        await store.putTable({ diagramId, table: remoteTable });
                    }
                }

                // Compare relationships
                const localRelationships = await store.listRelationships(diagramId);
                for (const remoteRel of remoteData.relationships) {
                    const localRel = localRelationships.find(r => r.id === remoteRel.id);
                    if (!localRel || remoteRel.createdAt > localRel.createdAt) {
                        if (!localRel) {
                            await store.addRelationship({ diagramId, relationship: remoteRel });
                        } else {
                            await store.updateRelationship({ id: remoteRel.id, attributes: remoteRel });
                        }
                    }
                }

                // Compare dependencies
                const localDependencies = await store.listDependencies(diagramId);
                for (const remoteDep of remoteData.dependencies) {
                    const localDep = localDependencies.find(d => d.id === remoteDep.id);
                    if (!localDep || remoteDep.createdAt > localDep.createdAt) {
                        if (!localDep) {
                            await store.addDependency({ diagramId, dependency: remoteDep });
                        } else {
                            await store.updateDependency({ id: remoteDep.id, attributes: remoteDep });
                        }
                    }
                }

                // Finally update context to reflect new state
                loadDiagram(diagramId);
            }
        } catch (error) {
            console.error('Failed to sync diagram:', error);
        } finally {
            isSyncingRef.current = false;
            setIsSyncing(false);
        }
    }, [store, loadDiagram]);

    events.useSubscription(handleLoadDiagram as any);

    return (
        <syncContext.Provider value={{ isSyncing }}>
            {children}
        </syncContext.Provider>
    );
};
