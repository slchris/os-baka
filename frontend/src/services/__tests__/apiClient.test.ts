/**
 * API Client Tests — Extended
 * Tests for HttpClient error mapping, type shapes, and response handling
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

      const result = mapApiError(axiosError);
      expect(result.message).toBe('Invalid credentials');
    });

    it('should extract message field from backend response', () => {
      const axiosError = {
        isAxiosError: true,
        response: {
          status: 400,
          data: { message: 'Validation failed' },
        },
        message: 'Request failed with status code 400',
      };

      const result = mapApiError(axiosError);
      expect(result.message).toBe('Validation failed');
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

    it('should handle null/undefined errors', () => {
      const result = mapApiError(null);
      expect(result.message).toBe('Unexpected error');
    });

    it('should return status from axios errors', () => {
      const axiosError = {
        isAxiosError: true,
        response: {
          status: 403,
          data: { error: 'Forbidden' },
        },
        message: 'Forbidden',
      };

      const result = mapApiError(axiosError);
      expect(result.status).toBe(403);
    });

    it('should handle network errors without response', () => {
      const axiosError = {
        isAxiosError: true,
        message: 'Network Error',
      };

      const result = mapApiError(axiosError);
      // Without response, falls through to generic Error
      expect(result.message).toBeTruthy();
    });

    it('should handle 5xx server errors', () => {
      const axiosError = {
        isAxiosError: true,
        response: {
          status: 500,
          data: { error: 'Internal Server Error' },
        },
        message: 'Request failed',
      };

      const result = mapApiError(axiosError);
      expect(result.status).toBe(500);
      expect(result.message).toBe('Internal Server Error');
    });

    it('should handle 429 rate limit errors', () => {
      const axiosError = {
        isAxiosError: true,
        response: {
          status: 429,
          data: { error: 'Rate limit exceeded' },
        },
        message: 'Too Many Requests',
      };

      const result = mapApiError(axiosError);
      expect(result.status).toBe(429);
      expect(result.message).toBe('Rate limit exceeded');
    });
  });
});
