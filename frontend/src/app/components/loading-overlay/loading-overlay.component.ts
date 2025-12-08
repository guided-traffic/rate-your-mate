import { Component, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AuthService } from '../../services/auth.service';
import { ConnectionStatusService } from '../../services/connection-status.service';

@Component({
  selector: 'app-loading-overlay',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="overlay" [class.hidden]="!showSpinner()">
      <div class="overlay-content">
        <div class="spinner"></div>
        <p class="message">{{ message() }}</p>
      </div>
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .overlay {
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background: rgba(15, 15, 15, 0.95);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 10000;
      backdrop-filter: blur(4px);
      transition: opacity 0.3s ease;
    }

    .overlay.hidden {
      opacity: 0;
      pointer-events: none;
    }

    .overlay-content {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 24px;
    }

    .spinner {
      width: 48px;
      height: 48px;
      border: 4px solid $bg-tertiary;
      border-top-color: $accent-primary;
      border-radius: 50%;
      animation: spin 1s linear infinite;
    }

    @keyframes spin {
      to {
        transform: rotate(360deg);
      }
    }

    .message {
      color: $text-secondary;
      font-size: 1rem;
      text-align: center;
      margin: 0;
      animation: pulse 2s ease-in-out infinite;
    }

    @keyframes pulse {
      0%, 100% {
        opacity: 1;
      }
      50% {
        opacity: 0.6;
      }
    }
  `]
})
export class LoadingOverlayComponent {
  private authService = inject(AuthService);
  private connectionStatus = inject(ConnectionStatusService);

  showSpinner = computed(() => {
    const hasUser = !!this.authService.user();
    const isAuthenticated = this.authService.isAuthenticated();
    const isDisconnected = this.connectionStatus.isDisconnected();

    // Show spinner if: authenticated but still loading user data OR disconnected from backend
    return (isAuthenticated && !hasUser) || isDisconnected;
  });

  message = computed(() => {
    if (this.connectionStatus.isDisconnected()) {
      const attempt = this.connectionStatus.currentReconnectAttempt();
      return attempt > 0
        ? `Backend connection lost. Reconnecting... (Attempt ${attempt})`
        : 'Backend connection lost. Reconnecting...';
    }
    return 'Loading...';
  });
}
