import { createContext } from 'react';

export interface SyncContext {
    isSyncing: boolean;
}

export const syncContext = createContext<SyncContext>({
    isSyncing: false,
});
