/**
 * src/test/server.js
 *
 * MSW Node server (Vitest runs in Node, not a browser).
 * Import `server` in setup.js; import `handlers` here to compose mocks.
 */
import { setupServer } from 'msw/node'
import { handlers } from './handlers.js'

export const server = setupServer(...handlers)
