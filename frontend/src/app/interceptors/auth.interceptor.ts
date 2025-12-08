import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { AuthService } from '../services/auth.service';
import { ConnectionStatusService } from '../services/connection-status.service';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  const router = inject(Router);
  const connectionStatus = inject(ConnectionStatusService);

  const token = authService.getToken();

  if (token) {
    req = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }

  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      console.log('[AuthInterceptor] Error:', error.status, req.url);

      if (error.status === 401) {
        console.log('[AuthInterceptor] 401 - Removing token and redirecting to login');
        authService.removeToken();
        router.navigate(['/login']);
      } else if (error.status === 0 || error.status >= 500) {
        // Network error or server error - backend is unavailable
        console.log('[AuthInterceptor] Backend unavailable:', error.status);
        connectionStatus.setDisconnected();
      }

      return throwError(() => error);
    })
  );
};
