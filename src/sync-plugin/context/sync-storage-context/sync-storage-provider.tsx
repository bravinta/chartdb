import React, { useContext } from 'react';
import { StorageProvider } from '../../../context/storage-context/storage-provider';
import { storageContext } from '../../../context/storage-context/storage-context';

export const SyncStorageProviderInner: React.FC<React.PropsWithChildren> = ({
    children,
}) => {
    const baseContext = useContext(storageContext);

    return (
        <storageContext.Provider value={baseContext}>
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
