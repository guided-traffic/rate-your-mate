import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';

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
export class LoginComponent implements OnInit {
  private auth = inject(AuthService);
  private router = inject(Router);

  ngOnInit(): void {
    // Redirect to games page if already logged in
    if (this.auth.getToken()) {
      this.router.navigate(['/games']);
    }
  }

  login(): void {
    this.auth.login();
  }
}
