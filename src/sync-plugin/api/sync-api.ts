import { SyncPushRequest, SyncPullResponse, SyncErrorResponse } from '../types/sync-types';

export const syncApi = {
    pull: async (diagramId: string): Promise<SyncPullResponse> => {
        const apiUrl = import.meta.env.VITE_SYNC_API_URL;
        const apiSecret = import.meta.env.API_SECRET; // from AGENTS.md

        if (!apiUrl) {
            throw new Error('VITE_SYNC_API_URL is not defined in environment variables.');
        }

        const response = await fetch(`${apiUrl}/api/sync/pull/${diagramId}`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Secret': apiSecret || '',
            },
        });

        if (!response.ok) {
            let errorMessage = `Failed to pull diagram ${diagramId}. Status: ${response.status}`;
            try {
                const errorData = await response.json() as SyncErrorResponse;
                if (errorData.error) {
                    errorMessage = errorData.error;
                }
            } catch (e) {
                // Ignore JSON parse error
            }
            throw new Error(errorMessage);
        }

        return await response.json() as SyncPullResponse;
    },

    push: async (request: SyncPushRequest): Promise<void> => {
        const apiUrl = import.meta.env.VITE_SYNC_API_URL;
        const apiSecret = import.meta.env.API_SECRET; // from AGENTS.md

        if (!apiUrl) {
            throw new Error('VITE_SYNC_API_URL is not defined in environment variables.');
        }

        const response = await fetch(`${apiUrl}/api/sync/push`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-API-Secret': apiSecret || '',
            },
            body: JSON.stringify(request),
        });

        if (!response.ok) {
            let errorMessage = `Failed to push sync changes. Status: ${response.status}`;
            try {
                const errorData = await response.json() as SyncErrorResponse;
                if (errorData.error) {
                    errorMessage = errorData.error;
                }
            } catch (e) {
                // Ignore JSON parse error
            }
            throw new Error(errorMessage);
        }
    }
};
