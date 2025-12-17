import { Component, inject, OnInit, OnDestroy, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { SettingsService } from '../../services/settings.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="login-page">
      <div class="login-container">
        <div class="login-header">
          <div class="logo">
            <img src="game-controller.png" alt="Rate your Mate" class="logo-icon">
            <h1 class="logo-text">Rate your Mate</h1>
          </div>
          <p class="tagline">Gemeinsam eine gute Zeit haben.</p>
        </div>

        <!-- Countdown Section -->
        @if (hasCountdown() && !isCountdownExpired()) {
          <div class="countdown-card">
            <div class="countdown-header">
              <span class="countdown-icon">⏰</span>
              <h3>Countdown zur LAN-Party</h3>
            </div>
            <div class="countdown-timer">
              <div class="countdown-unit">
                <span class="countdown-value">{{ countdownDays() }}</span>
                <span class="countdown-label">Tage</span>
              </div>
              <span class="countdown-separator">:</span>
              <div class="countdown-unit">
                <span class="countdown-value">{{ countdownHours() }}</span>
                <span class="countdown-label">Stunden</span>
              </div>
              <span class="countdown-separator">:</span>
              <div class="countdown-unit">
                <span class="countdown-value">{{ countdownMinutes() }}</span>
                <span class="countdown-label">Minuten</span>
              </div>
              <span class="countdown-separator">:</span>
              <div class="countdown-unit">
                <span class="countdown-value">{{ countdownSeconds() }}</span>
                <span class="countdown-label">Sekunden</span>
              </div>
            </div>
          </div>
        }

        <div class="login-card">
          <h2>Willkommen!</h2>
          <p class="description">
            Melde dich mit deinem Steam-Account an, um gemeinsam mit deinen Freunden zu spielen.
          </p>

          <button class="btn btn-steam" (click)="login()">
            <img src="logos/steam.png" alt="Steam" class="steam-icon">
            Mit Steam anmelden
          </button>

          <div class="features">
            <div class="feature">
              <span class="feature-icon">🎮</span>
              <span>Findet gemeinsame Games schneller</span>
            </div>
            <div class="feature">
              <span class="feature-icon">⭐</span>
              <span>Bewertet euch gegenseitig</span>
            </div>
            <div class="feature">
              <span class="feature-icon">🏆</span>
              <span>Findet den besten Spieler unter euch</span>
            </div>
            <div class="feature">
              <span class="feature-icon">💬</span>
              <span>Gemeinsamer Chat</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .login-page {
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 24px;
      background: radial-gradient(ellipse at top, #1a1a2e 0%, $bg-primary 100%);
    }

    .login-container {
      max-width: 480px;
      width: 100%;
    }

    .login-header {
      text-align: center;
      margin-bottom: 32px;
    }

    .logo {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 12px;
      margin-bottom: 12px;

      .logo-icon {
        width: 48px;
        height: 48px;
      }

      .logo-text {
        font-size: 32px;
        font-weight: 700;
        background: $gradient-primary;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
      }
    }

    .tagline {
      color: $text-secondary;
      font-size: 18px;
    }

    .countdown-card {
      background: linear-gradient(135deg, rgba($accent-primary, 0.15), rgba($accent-secondary, 0.15));
      border: 1px solid rgba($accent-primary, 0.3);
      border-radius: $radius-xl;
      padding: 24px;
      margin-bottom: 24px;
      text-align: center;

      &.expired {
        background: linear-gradient(135deg, rgba($accent-success, 0.15), rgba($accent-primary, 0.15));
        border-color: rgba($accent-success, 0.3);
        animation: pulse 2s ease-in-out infinite;
      }
    }

    .countdown-header {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      margin-bottom: 16px;

      .countdown-icon {
        font-size: 24px;
      }

      h3 {
        font-size: 18px;
        font-weight: 600;
        margin: 0;
        background: $gradient-primary;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
      }
    }

    .countdown-message {
      font-size: 16px;
      color: $accent-success;
      font-weight: 500;
    }

    .countdown-timer {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
    }

    .countdown-unit {
      display: flex;
      flex-direction: column;
      align-items: center;
      min-width: 60px;
    }

    .countdown-value {
      font-size: 32px;
      font-weight: 700;
      font-family: 'JetBrains Mono', monospace;
      background: $gradient-primary;
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
      line-height: 1;
    }

    .countdown-label {
      font-size: 11px;
      color: $text-muted;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-top: 4px;
    }

    .countdown-separator {
      font-size: 24px;
      font-weight: 700;
      color: $text-muted;
      margin-bottom: 16px;
    }

    @keyframes pulse {
      0%, 100% {
        transform: scale(1);
        box-shadow: 0 0 0 0 rgba($accent-success, 0.4);
      }
      50% {
        transform: scale(1.01);
        box-shadow: 0 0 20px 5px rgba($accent-success, 0.2);
      }
    }

    .login-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-xl;
      padding: 40px;
      text-align: center;

      h2 {
        font-size: 24px;
        margin-bottom: 12px;
      }

      .description {
        color: $text-secondary;
        margin-bottom: 32px;
        line-height: 1.6;
      }
    }

    .btn-steam {
      width: 100%;
      padding: 16px 24px;
      font-size: 18px;
      background: linear-gradient(135deg, #1b2838, #2a475e);
      border: none;
      border-radius: $radius-md;
      color: white;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 12px;
      transition: all $transition-fast;

      &:hover {
        background: linear-gradient(135deg, #2a475e, #3a5a7c);
        transform: translateY(-2px);
        box-shadow: $shadow-lg;
      }

      .steam-icon {
        height: 42px;
        width: auto;
      }
    }

    .features {
      margin-top: 32px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .feature {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 16px;
      background: $bg-tertiary;
      border-radius: $radius-md;
      font-size: 14px;

      .feature-icon {
        font-size: 20px;
      }
    }
  `]
})
export class LoginComponent implements OnInit, OnDestroy {
  private auth = inject(AuthService);
  private router = inject(Router);
  private settingsService = inject(SettingsService);

  private countdownInterval: ReturnType<typeof setInterval> | null = null;
  private countdownTargetTime = signal<Date | null>(null);
  private currentTime = signal<Date>(new Date());

  // Computed signals for countdown display
  hasCountdown = computed(() => this.countdownTargetTime() !== null);
  isCountdownExpired = computed(() => {
    const target = this.countdownTargetTime();
    if (!target) return false;
    return this.currentTime() >= target;
  });

  private remainingMs = computed(() => {
    const target = this.countdownTargetTime();
    if (!target) return 0;
    const diff = target.getTime() - this.currentTime().getTime();
    return Math.max(0, diff);
  });

  countdownDays = computed(() => {
    const ms = this.remainingMs();
    return String(Math.floor(ms / (1000 * 60 * 60 * 24))).padStart(2, '0');
  });

  countdownHours = computed(() => {
    const ms = this.remainingMs();
    return String(Math.floor((ms % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))).padStart(2, '0');
  });

  countdownMinutes = computed(() => {
    const ms = this.remainingMs();
    return String(Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60))).padStart(2, '0');
  });

  countdownSeconds = computed(() => {
    const ms = this.remainingMs();
    return String(Math.floor((ms % (1000 * 60)) / 1000)).padStart(2, '0');
  });

  ngOnInit(): void {
    // Redirect to games page if already logged in
    if (this.auth.getToken()) {
      this.router.navigate(['/games']);
      return;
    }

    // Load countdown from server (public endpoint, no auth required)
    this.loadCountdown();

    // Update current time every second for countdown
    this.countdownInterval = setInterval(() => {
      this.currentTime.set(new Date());
    }, 1000);
  }

  ngOnDestroy(): void {
    if (this.countdownInterval) {
      clearInterval(this.countdownInterval);
    }
  }

  private loadCountdown(): void {
    this.settingsService.getPublicCountdown().subscribe({
      next: (response) => {
        if (response.countdown_target) {
          const targetDate = new Date(response.countdown_target);
          this.countdownTargetTime.set(targetDate);
        }
      },
      error: (err) => {
        console.error('Failed to load countdown:', err);
      }
    });
  }

  login(): void {
    this.auth.login();
  }
}
