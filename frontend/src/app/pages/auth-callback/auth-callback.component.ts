import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, ActivatedRoute } from '@angular/router';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-auth-callback',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="callback-page">
      <div class="callback-container">
        @if (error) {
          <div class="error-state">
            <span class="error-icon">❌</span>
            <h2>Anmeldung fehlgeschlagen</h2>
            <p>{{ error }}</p>
            <button class="btn btn-primary" (click)="goToLogin()">Zurück zum Login</button>
          </div>
        } @else {
          <div class="loading-state">
            <div class="spinner-large"></div>
            <h2>Anmeldung wird verarbeitet...</h2>
            <p>Du wirst gleich weitergeleitet.</p>
          </div>
        }
      </div>
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .callback-page {
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
      background: $bg-primary;
    }

    .callback-container {
      text-align: center;
    }

    .loading-state, .error-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 16px;

      h2 {
        font-size: 24px;
        margin: 0;
      }

      p {
        color: $text-secondary;
        margin: 0;
      }
    }

    .spinner-large {
      width: 48px;
      height: 48px;
      border: 3px solid $border-color;
      border-top-color: $accent-primary;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    .error-icon {
      font-size: 48px;
    }

    @keyframes spin {
      to {
        transform: rotate(360deg);
      }
    }
  `]
})
export class AuthCallbackComponent implements OnInit {
  private auth = inject(AuthService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);

  error: string | null = null;

  ngOnInit(): void {
    const params = this.route.snapshot.queryParams;

    if (params['error']) {
      this.error = params['error'];
      return;
    }

    const token = params['token'];
    if (token) {
      this.auth.handleCallback(token);

      // Navigate to games after successful login
      // WebSocket connection is handled by App component
      setTimeout(() => {
        this.router.navigate(['/games']);
      }, 500);
    } else {
      this.error = 'Kein Token erhalten. Bitte versuche es erneut.';
    }
  }

  goToLogin(): void {
    this.router.navigate(['/login']);
  }
}
