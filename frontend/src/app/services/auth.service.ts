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
    // Check for existing token on startup
    const token = this.getToken();
    console.log('[AuthService] Constructor - Token exists:', !!token);
    this.tokenExists.set(!!token);
    if (token) {
      // Set loading to true BEFORE loadCurrentUser to prevent race condition
      this.loading.set(true);
      this.loadCurrentUser();
    }
  }

  getToken(): string | null {
    return localStorage.getItem(this.TOKEN_KEY);
  }

  setToken(token: string): void {
    localStorage.setItem(this.TOKEN_KEY, token);
    this.tokenExists.set(true);
  }

  removeToken(): void {
    localStorage.removeItem(this.TOKEN_KEY);
    this.tokenExists.set(false);
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
    console.log('[AuthService] loadCurrentUser - Starting...');
    this.loading.set(true);
    this.http.get<{ user: CurrentUser }>(`${environment.apiUrl}/auth/me`).subscribe({
      next: (response) => {
        console.log('[AuthService] loadCurrentUser - Success:', response.user.username);
        this.currentUser.set(response.user);
        this.loading.set(false);
      },
      error: (error) => {
        console.error('[AuthService] loadCurrentUser - Error:', error.status, error.message);
        this.loading.set(false);

        // Only clear token and redirect for authentication errors (401)
        // The interceptor already handles 401, but we check here too for safety
        if (error.status === 401) {
          console.log('[AuthService] loadCurrentUser - 401, removing token');
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
