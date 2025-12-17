import { Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { environment } from '../../environments/environment';
import { CurrentUser } from '../models/user.model';

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly TOKEN_KEY = 'lan_party_token';

  private currentUser = signal<CurrentUser | null>(null);
  private loading = signal(false);
  private tokenExists = signal(false);

  readonly user = this.currentUser.asReadonly();
  // isAuthenticated is true if we have a token (even while loading user data)
  readonly isAuthenticated = computed(() => this.tokenExists() || !!this.currentUser());
  readonly isLoading = this.loading.asReadonly();
  readonly credits = computed(() => this.currentUser()?.credits ?? 0);

  constructor(
    private http: HttpClient,
    private router: Router
  ) {
    // Check for existing token on startup (getToken() automatically removes expired tokens)
    const token = this.getToken();
    this.tokenExists.set(!!token);
    if (token) {
      // Set loading to true BEFORE loadCurrentUser to prevent race condition
      this.loading.set(true);
      this.loadCurrentUser();
    } else {
      // No valid token - ensure clean state
      this.currentUser.set(null);
    }
  }

  getToken(): string | null {
    const token = localStorage.getItem(this.TOKEN_KEY);
    if (token && this.isTokenExpired(token)) {
      console.log('[AuthService] Token is expired, removing it');
      this.removeToken();
      return null;
    }
    return token;
  }

  /**
   * Check if a JWT token is expired by decoding the payload and checking the exp claim.
   * Returns true if the token is expired or invalid.
   */
  isTokenExpired(token: string): boolean {
    try {
      const payload = this.decodeTokenPayload(token);
      if (!payload || !payload.exp) {
        console.log('[AuthService] Token has no exp claim');
        return true;
      }
      // exp is in seconds, Date.now() is in milliseconds
      const expirationTime = payload.exp * 1000;
      const now = Date.now();
      const isExpired = now >= expirationTime;

      if (isExpired) {
        const expiredAgo = Math.round((now - expirationTime) / 1000 / 60);
        console.log(`[AuthService] Token expired ${expiredAgo} minutes ago`);
      } else {
        const expiresIn = Math.round((expirationTime - now) / 1000 / 60);
        console.log(`[AuthService] Token expires in ${expiresIn} minutes`);
      }

      return isExpired;
    } catch (error) {
      console.error('[AuthService] Error checking token expiry:', error);
      return true;
    }
  }

  /**
   * Decode the payload of a JWT token without verifying the signature.
   * This is safe for client-side expiry checks since the server will still validate.
   */
  private decodeTokenPayload(token: string): { exp?: number; uid?: number; name?: string; steam_id?: string } | null {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) {
        return null;
      }
      // Decode base64url to base64, then decode
      const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
      const jsonPayload = decodeURIComponent(
        atob(base64)
          .split('')
          .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
          .join('')
      );
      return JSON.parse(jsonPayload);
    } catch {
      return null;
    }
  }

  setToken(token: string): void {
    localStorage.setItem(this.TOKEN_KEY, token);
    this.tokenExists.set(true);
  }

  removeToken(): void {
    localStorage.removeItem(this.TOKEN_KEY);
    this.tokenExists.set(false);
    this.currentUser.set(null);
  }

  login(): void {
    // Redirect to backend Steam auth endpoint
    window.location.href = `${environment.apiUrl}/auth/steam`;
  }

  handleCallback(token: string): void {
    this.setToken(token);
    this.loadCurrentUser();
  }

  logout(): void {
    this.http.post(`${environment.apiUrl}/auth/logout`, {}).subscribe({
      complete: () => {
        this.removeToken();
        this.currentUser.set(null);
        this.router.navigate(['/login']);
      },
      error: () => {
        // Even if the API call fails, clear local state
        this.removeToken();
        this.currentUser.set(null);
        this.router.navigate(['/login']);
      }
    });
  }

  loadCurrentUser(): void {
    this.loading.set(true);
    this.http.get<{ user: CurrentUser }>(`${environment.apiUrl}/auth/me`).subscribe({
      next: (response) => {
        this.currentUser.set(response.user);
        this.loading.set(false);
      },
      error: (error) => {
        console.error('[AuthService] loadCurrentUser - Error:', error.status, error.message);
        this.loading.set(false);

        // Clear token and redirect for authentication errors:
        // - 401: Invalid/expired token
        // - 404: User not found (e.g., DB was reset but client still has old token)
        if (error.status === 401 || error.status === 404) {
          console.log(`[AuthService] loadCurrentUser - ${error.status}, removing token and redirecting to login`);
          this.removeToken();
          this.currentUser.set(null);
          this.router.navigate(['/login']);
        }
        // For other errors (network, server down, etc.), keep the token
        // and let the user retry or the app recover
      }
    });
  }

  checkAuth(): void {
    if (this.getToken()) {
      this.loadCurrentUser();
    }
  }

  updateCredits(credits: number): void {
    const user = this.currentUser();
    if (user) {
      this.currentUser.set({ ...user, credits });
    }
  }

  refreshUser(): void {
    this.loadCurrentUser();
  }
}
