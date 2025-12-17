import { HttpInterceptorFn, HttpErrorResponse, HttpResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, tap, throwError } from 'rxjs';
import { ConnectionStatusService } from '../services/connection-status.service';
import { LatencyService } from '../services/latency.service';

const TOKEN_KEY = 'lan_party_token';

/**
 * Get token directly from localStorage to avoid circular dependency.
 * AuthService constructor makes HTTP requests, which would trigger this interceptor,
 * which would inject AuthService - causing a circular dependency (NG0200).
 */
function getTokenFromStorage(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

/**
 * Remove token and redirect to login on 401.
 * We do this directly instead of through AuthService to avoid circular dependency.
 */
function handleUnauthorized(router: Router): void {
  localStorage.removeItem(TOKEN_KEY);
  router.navigate(['/login']);
}

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const router = inject(Router);
  const connectionStatus = inject(ConnectionStatusService);
  const latencyService = inject(LatencyService);

  const token = getTokenFromStorage();
  const startTime = performance.now();

  if (token) {
    req = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }

  return next(req).pipe(
    tap(event => {
      if (event instanceof HttpResponse) {
        const latency = Math.round(performance.now() - startTime);
        latencyService.recordLatency(latency);
      }
    }),
    catchError((error: HttpErrorResponse) => {
      console.log('[AuthInterceptor] Error:', error.status, req.url);

      if (error.status === 401) {
        console.log('[AuthInterceptor] 401 - Removing token and redirecting to login');
        handleUnauthorized(router);
      } else if (error.status === 0 || error.status >= 500) {
        // Network error or server error - backend is unavailable
        console.log('[AuthInterceptor] Backend unavailable:', error.status);
        connectionStatus.setDisconnected();
      }

      return throwError(() => error);
    })
  );
};
