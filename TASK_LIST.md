TASK_LIST.md

Objetivo principal:
Implementar el sync plugin de ChartDB aislado en src/sync-plugin/ y la API REST
en sync-api/ usando Go, Fiber y GORM, siguiendo la arquitectura descrita en
AGENTS.md, sin romper compatibilidad con upstream.

Tareas pendientes:
- [ ] TASK-002: Crear cliente HTTP API en `src/sync-plugin/api/sync-api.ts` para comunicarse con la API remota (push/pull).
- [ ] TASK-003: Implementar la base de `SyncStorageProvider` (`src/sync-plugin/context/sync-storage-context/sync-storage-provider.tsx`) que envuelva a `StorageProvider` original e implemente su interfaz.
- [ ] TASK-004: Completar lógica de `SyncStorageProvider` añadiendo intercepción de operaciones CRUD y la estrategia Last-Write-Wins usando la API.
- [ ] TASK-005: Crear `SyncContext` y `SyncProvider` en `src/sync-plugin/context/sync-context/` para escuchar los eventos de la aplicación y ejecutar el pull inicial.
- [ ] TASK-006: Actualizar `src/lib/env.ts` para exponer `VITE_SYNC_API_URL` y `VITE_SYNC_ENABLED`.
- [ ] TASK-007: Modificar condicionalmente `src/pages/editor-page/editor-page.tsx` para reemplazar `StorageProvider` con `SyncStorageProvider` basado en la variable de entorno.
- [ ] TASK-008: Configurar proyecto Go en `sync-api/` (go mod init, dependencias Fiber, GORM) y crear la estructura base.
- [ ] TASK-009: Implementar los modelos de base de datos con GORM en `sync-api/models/` para diagramas, tablas, relaciones y dependencias.
- [ ] TASK-010: Implementar middleware de autenticación (verificando X-API-Secret) en `sync-api/middleware/`.
- [ ] TASK-011: Implementar handlers de la API en Go en `sync-api/handlers/` para GET `/api/sync/pull/:diagramId`.
- [ ] TASK-012: Implementar handlers de la API en Go en `sync-api/handlers/` para POST `/api/sync/push` con manejo de conflictos.
- [ ] TASK-013: Conectar la BD y montar la aplicación Fiber completa en `sync-api/main.go`.
- [ ] TASK-014: Crear `Dockerfile` y `docker-compose.yml` para desplegar la API en Go con PostgreSQL.
- [ ] TASK-015: Escribir y ejecutar pruebas de integración para el flujo de push/pull completo.

Tareas bloqueadas:
(tareas que dependen de otras o necesitan decision externa)

Tareas completadas:
- [x] TASK-001: Definir los tipos TypeScript para sincronización en `src/sync-plugin/types/sync-types.ts` (interfaces de peticiones/respuestas pull/push).