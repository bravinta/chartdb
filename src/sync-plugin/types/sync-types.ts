import type { Diagram } from '../../lib/domain/diagram';
import type { DBTable } from '../../lib/domain/db-table';
import type { DBRelationship } from '../../lib/domain/db-relationship';
import type { DBDependency } from '../../lib/domain/db-dependency';

export interface SyncPushRequest {
    diagram: Diagram;
    tables: DBTable[];
    relationships: DBRelationship[];
    dependencies: DBDependency[];
}

export interface SyncPullResponse {
    diagram: Diagram;
    tables: DBTable[];
    relationships: DBRelationship[];
    dependencies: DBDependency[];
}

export interface SyncErrorResponse {
    error: string;
}
