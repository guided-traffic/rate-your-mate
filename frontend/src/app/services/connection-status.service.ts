import { Injectable, signal, computed, inject, OnDestroy } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../environments/environment';

export type ConnectionState = 'loading' | 'connected' | 'disconnected' | 'reconnecting';

@Injectable({
  providedIn: 'root'
})
export class ConnectionStatusService implements OnDestroy {
  private http = inject(HttpClient);

  private state = signal<ConnectionState>('loading');
  private initialLoadComplete = signal(false);
  private reconnectAttempt = signal(0);
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private isCheckingHealth = false;
  private readonly maxReconnectAttempts = 10;
  private readonly reconnectBaseDelay = 2000;

  readonly isDisconnected = computed(() => this.state() === 'disconnected' || this.state() === 'reconnecting');
  readonly currentReconnectAttempt = this.reconnectAttempt.asReadonly();

  ngOnDestroy(): void {
    this.clearReconnectTimer();
  }

  setConnected(): void {
    this.clearReconnectTimer();
    this.state.set('connected');
    this.reconnectAttempt.set(0);
  }

  setDisconnected(): void {
    if (this.state() !== 'disconnected' && this.state() !== 'reconnecting') {
      this.state.set('disconnected');
      this.startReconnect();
    }
  }

  markInitialLoadComplete(): void {
    this.initialLoadComplete.set(true);
    if (this.state() === 'loading') {
      this.state.set('connected');
    }
  }

  private startReconnect(): void {
    // Don't start another reconnect if already in progress
    if (this.reconnectTimer || this.isCheckingHealth) {
      console.log('[ConnectionStatus] Reconnect already in progress, skipping');
      return;
    }

    this.attemptReconnect();
  }

  private attemptReconnect(): void {
    // Double-check we're not already reconnecting
    if (this.reconnectTimer || this.isCheckingHealth) {
      return;
    }

    const attempt = this.reconnectAttempt() + 1;

    if (attempt > this.maxReconnectAttempts) {
      console.log('[ConnectionStatus] Max reconnect attempts reached');
      this.state.set('disconnected');
      return;
    }

    this.state.set('reconnecting');
    this.reconnectAttempt.set(attempt);

    // Exponential backoff with max of 10 seconds
    const delay = Math.min(this.reconnectBaseDelay * Math.pow(1.5, attempt - 1), 10000);
    console.log(`[ConnectionStatus] Reconnect attempt ${attempt} in ${delay}ms`);

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.checkBackendHealth();
    }, delay);
  }

  private checkBackendHealth(): void {
    // Prevent multiple health checks running simultaneously
    if (this.isCheckingHealth) {
      return;
    }

    this.isCheckingHealth = true;

    // Use the health check endpoint under /api/v1 for proper ingress routing
    this.http.get(`${environment.apiUrl}/health`, { responseType: 'text' }).subscribe({
      next: () => {
        console.log('[ConnectionStatus] Backend is back online');
        this.isCheckingHealth = false;
        this.setConnected();
        // Trigger a page reload to refresh all data
        window.location.reload();
      },
      error: () => {
        console.log('[ConnectionStatus] Backend still unavailable');
        this.isCheckingHealth = false;
        this.attemptReconnect();
      }
    });
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}
