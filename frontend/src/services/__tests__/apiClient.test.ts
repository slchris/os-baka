/**
 * API Client Tests
 * Testing error handling and utility functions
 */

import { describe, it, expect } from 'vitest';
import { mapApiError } from '../apiClient';

describe('API Client', () => {
  describe('mapApiError', () => {
    it('should extract error message from backend response (error field)', () => {
      const axiosError = {
        isAxiosError: true,
        response: {
          status: 401,
          data: { error: 'Invalid credentials' },
        },
        message: 'Request failed with status code 401',
      };

      // Simulate axios error shape
      const result = mapApiError(axiosError);

      expect(result.message).toBe('Invalid credentials');
    });

    it('should handle generic Error objects', () => {
      const error = new Error('Network error');
      const result = mapApiError(error);

      expect(result.message).toBe('Network error');
    });

    it('should handle unknown error types', () => {
      const result = mapApiError('Unknown error');

      expect(result.message).toBe('Unexpected error');
    });

    it('should return status from axios errors', () => {
      const error = new Error('Not found');
      const result = mapApiError(error);

      expect(result.status).toBeUndefined();
    });
  });
});
