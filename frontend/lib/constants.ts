/**
 * Shared constants for the frontend application.
 * BACKEND_URL is the Go backend server that the Next.js API routes proxy to.
 */
export const BACKEND_URL =
  process.env.BACKEND_URL || "http://127.0.0.1:8000"

/** Maximum allowed request body size for test-message route (bytes). */
export const MAX_TEST_MESSAGE_BODY_SIZE = 32 * 1024 // 32 KB

/** Allowed model name pattern for test-message route. */
export const MODEL_NAME_PATTERN = /^[a-zA-Z0-9._-]{1,128}$/

/** Maximum message content length for test-message route (characters). */
export const MAX_MESSAGE_CONTENT_LENGTH = 4096

/** Maximum number of messages allowed in a single test-message request. */
export const MAX_MESSAGES_COUNT = 5
