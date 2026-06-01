/**
 * src/test/setup.js
 *
 * Global Vitest setup file.
 * - Starts the MSW server before all tests (Node request interception mode)
 * - Resets handlers after each test so tests don't bleed into each other
 * - Closes the server after all suites finish
 */
import { beforeAll, afterEach, afterAll } from 'vitest'
import { server } from './server.js'

beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
