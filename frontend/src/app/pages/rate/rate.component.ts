import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { UserService } from '../../services/user.service';
import { AchievementService } from '../../services/achievement.service';
import { VoteService } from '../../services/vote.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { SettingsService } from '../../services/settings.service';
import { SoundService } from '../../services/sound.service';
import { User } from '../../models/user.model';
import { Achievement } from '../../models/achievement.model';

@Component({
  selector: 'app-rate',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="page fade-in">
      <div class="page-header">
        <h1 class="page-title">Spieler bewerten</h1>
        <p class="page-subtitle">Wähle einen Spieler und ein Achievement aus</p>
      </div>

      @if (votingPaused()) {
        <div class="paused-banner">
          <span class="paused-icon">⏸️</span>
          <div class="paused-text">
            <strong>Voting ist pausiert</strong>
            <p>Der Admin hat das Voting vorübergehend deaktiviert. Bitte warte, bis es wieder aktiviert wird.</p>
          </div>
        </div>
      }

      <!-- Step 1: Select Player -->
      <div class="section">
        <h2 class="section-title">
          <span class="step-number">1</span>
          Spieler auswählen
        </h2>

        @if (loadingUsers()) {
          <div class="loading-container">
            <div class="spinner"></div>
          </div>
        } @else if (users().length === 0) {
          <div class="empty-state">
            <span class="empty-state-icon">👥</span>
            <p class="empty-state-title">Keine anderen Spieler online</p>
            <p>Warte bis andere Spieler der LAN Party beitreten.</p>
          </div>
        } @else {
          <div class="players-grid">
            @for (user of users(); track user.id) {
              <div
                class="player-card"
                [class.selected]="selectedUser()?.id === user.id"
                [style.background-image]="'url(' + (user.avatar_url || user.avatar_small || '/assets/default-avatar.png') + ')'"
                (click)="selectUser(user)"
              >
                <a
                  class="add-friend-btn"
                  [href]="user.profile_url"
                  target="_blank"
                  (click)="$event.stopPropagation()"
                  title="Steam-Profil öffnen"
                >
                  <svg viewBox="0 0 24 24" fill="currentColor" class="steam-icon">
                    <path d="M12 2a10 10 0 0 1 10 10 10 10 0 0 1-10 10c-4.6 0-8.45-3.08-9.64-7.27l3.83 1.58a2.84 2.84 0 0 0 2.78 2.27c1.56 0 2.83-1.27 2.83-2.83v-.13l3.4-2.43h.08c2.08 0 3.77-1.69 3.77-3.77s-1.69-3.77-3.77-3.77-3.77 1.69-3.77 3.77v.05l-2.37 3.46-.16-.01c-.53 0-1.02.15-1.44.41L2.1 10.68A10 10 0 0 1 12 2m0 18a8 8 0 0 0 8-8 8 8 0 0 0-8-8 8 8 0 0 0-7.89 6.78l1.85.76a2.83 2.83 0 0 1 3.4-.72l1.99-1.47c-.02-.12-.02-.24-.02-.35 0-1.39 1.13-2.52 2.52-2.52S16.37 7.61 16.37 9s-1.13 2.52-2.52 2.52h-.19l-1.4 2.03c.08.27.12.55.12.84 0 1.39-1.13 2.52-2.52 2.52-.35 0-.69-.07-1-.21l-.83-.34A8 8 0 0 0 12 20m3.85-11a1.26 1.26 0 0 0-1.26 1.26c0 .7.56 1.26 1.26 1.26.7 0 1.26-.56 1.26-1.26 0-.7-.56-1.26-1.26-1.26m-7.16 5.21.69.29c.28.11.58.13.87.05a1.28 1.28 0 0 0 .87-1.58 1.28 1.28 0 0 0-1.16-.88l-.72-.3a1.57 1.57 0 0 0-.55 2.42z"/>
                  </svg>
                </a>
                <div class="player-name-overlay">
                  <span class="player-name">{{ user.username }}</span>
                </div>
              </div>
            }
          </div>
        }
      </div>

      <!-- Step 2: Select Achievement -->
      @if (selectedUser()) {
        <div class="section fade-in">
          <h2 class="section-title">
            <span class="step-number">2</span>
            Achievement auswählen für <span class="highlight">{{ selectedUser()!.username }}</span>
          </h2>

          <div class="achievements-section">
            <h3 class="achievement-category positive">
              <span>👍</span> Positive Achievements
            </h3>
            <div class="achievements-grid">
              @for (achievement of positiveAchievements(); track achievement.id) {
                <button
                  class="achievement-btn positive"
                  [class.selected]="selectedAchievement()?.id === achievement.id"
                  (click)="selectAchievement(achievement)"
                >
                  @if (achievement.image_url) {
                    <div class="achievement-icon-wrapper gold-dust">
                      <img [src]="achievement.image_url" [alt]="achievement.name" class="achievement-icon positive" />
                    </div>
                  }
                  <span class="achievement-name">{{ achievement.name }}</span>
                  <span class="achievement-desc">{{ achievement.description }}</span>
                </button>
              }
            </div>

            <h3 class="achievement-category negative">
              <span>👎</span> Negative Achievements
            </h3>
            <div class="achievements-grid">
              @for (achievement of negativeAchievements(); track achievement.id) {
                <button
                  class="achievement-btn negative"
                  [class.selected]="selectedAchievement()?.id === achievement.id"
                  (click)="selectAchievement(achievement)"
                >
                  @if (achievement.image_url) {
                    <div class="achievement-icon-wrapper shatter-dust">
                      <img [src]="achievement.image_url" [alt]="achievement.name" class="achievement-icon negative" />
                    </div>
                  }
                  <span class="achievement-name">{{ achievement.name }}</span>
                  <span class="achievement-desc">{{ achievement.description }}</span>
                </button>
              }
            </div>
          </div>
        </div>
      }

      <!-- Step 3: Confirm -->
      @if (selectedUser() && selectedAchievement()) {
        <div class="section confirm-section fade-in">
          <div class="confirm-card">
            <div class="confirm-preview">
              <img
                [src]="auth.user()?.avatar_small || '/assets/default-avatar.png'"
                class="avatar"
                alt="Du"
              />
              <span class="arrow">→</span>
              <img
                [src]="selectedUser()!.avatar_small || '/assets/default-avatar.png'"
                class="avatar"
                alt="{{ selectedUser()!.username }}"
              />
            </div>

            <p class="confirm-text">
              Du gibst <strong>{{ selectedUser()!.username }}</strong> das Achievement
              <span
                class="achievement-chip"
                [class.positive]="selectedAchievement()!.is_positive"
                [class.negative]="!selectedAchievement()!.is_positive"
              >
                {{ selectedAchievement()!.name }}
              </span>
            </p>

            <div class="points-selector">
              <label class="points-label">Punkte:</label>
              <div class="slider-container">
                <input
                  type="range"
                  min="1"
                  max="3"
                  step="1"
                  [ngModel]="selectedPoints()"
                  (ngModelChange)="selectedPoints.set($event)"
                  class="points-slider"
                  [class.disabled]="auth.credits() < 1"
                />
                <div class="points-markers">
                  <span [class.active]="selectedPoints() >= 1" [class.affordable]="auth.credits() >= 1">1</span>
                  <span [class.active]="selectedPoints() >= 2" [class.affordable]="auth.credits() >= 2">2</span>
                  <span [class.active]="selectedPoints() >= 3" [class.affordable]="auth.credits() >= 3">3</span>
                </div>
              </div>
              <span class="points-value">{{ selectedPoints() }} {{ selectedPoints() === 1 ? 'Punkt' : 'Punkte' }}</span>
            </div>

            <div class="secret-toggle">
              <label class="toggle-container" (click)="toggleSecret()">
                <span class="toggle-switch" [class.active]="isSecretVote()">
                  <span class="toggle-slider"></span>
                </span>
                <span class="toggle-label">
                  <span class="toggle-icon">{{ isSecretVote() ? '🕵️' : '👁️' }}</span>
                  {{ isSecretVote() ? 'Geheim abstimmen' : 'Offen abstimmen' }}
                </span>
              </label>
              <span class="toggle-hint">
                @if (isSecretVote()) {
                  Dein Name wird nicht angezeigt
                } @else {
                  Alle sehen, dass du abgestimmt hast
                }
              </span>
            </div>

            <button
              class="btn btn-primary btn-lg"
              [disabled]="submitting() || auth.credits() < selectedPoints() || votingPaused()"
              (click)="submitVote()"
            >
              @if (submitting()) {
                <div class="spinner"></div>
                Wird gesendet...
              } @else if (votingPaused()) {
                Voting pausiert
              } @else if (auth.credits() < selectedPoints()) {
                Nicht genug Credits
              } @else {
                Bewerten
              }
            </button>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .paused-banner {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 20px 24px;
      background: rgba(#f59e0b, 0.15);
      border: 2px solid #f59e0b;
      border-radius: $radius-lg;
      margin-bottom: 32px;

      .paused-icon {
        font-size: 32px;
      }

      .paused-text {
        strong {
          display: block;
          font-size: 16px;
          color: #f59e0b;
          margin-bottom: 4px;
        }

        p {
          font-size: 14px;
          color: $text-secondary;
          margin: 0;
        }
      }
    }

    .section {
      margin-bottom: 40px;
    }

    .section-title {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 20px;
      margin-bottom: 20px;

      .step-number {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 32px;
        height: 32px;
        background: $gradient-primary;
        border-radius: 50%;
        font-size: 16px;
        font-weight: 700;
      }

      .highlight {
        color: $accent-primary;
      }
    }

    .loading-container {
      display: flex;
      justify-content: center;
      padding: 48px;
    }

    .players-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
      gap: 16px;
    }

    .achievements-section {
      display: flex;
      flex-direction: column;
      gap: 24px;
    }

    .achievement-category {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 16px;
      font-weight: 600;

      &.positive {
        color: $accent-positive;
      }

      &.negative {
        color: $accent-negative;
      }
    }

    .achievements-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 16px;

      @media (max-width: 900px) {
        grid-template-columns: repeat(2, 1fr);
      }

      @media (max-width: 600px) {
        grid-template-columns: 1fr;
      }
    }

    .achievement-btn {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 12px;
      padding: 24px 20px;
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      cursor: pointer;
      text-align: center;
      transition: all $transition-fast;
      min-height: 160px;

      .achievement-icon-wrapper {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 72px;
        height: 72px;
        border-radius: $radius-md;
      }

      .achievement-icon {
        width: 56px;
        height: 56px;
        border-radius: $radius-sm;

        &.positive {
          filter: invert(65%) sepia(52%) saturate(5765%) hue-rotate(103deg) brightness(96%) contrast(85%);
        }

        &.negative {
          filter: invert(39%) sepia(95%) saturate(1834%) hue-rotate(336deg) brightness(96%) contrast(93%);
        }
      }

      &:hover {
        border-color: $border-light;
        transform: translateY(-2px);
      }

      &.positive {
        &:hover, &.selected {
          border-color: $accent-positive;
          background: rgba($accent-positive, 0.05);
        }
      }

      &.negative {
        &:hover, &.selected {
          border-color: $accent-negative;
          background: rgba($accent-negative, 0.05);
        }
      }

      &.selected {
        box-shadow: $shadow-glow;
        transform: translateY(-2px);
      }

      .achievement-name {
        font-weight: 700;
        font-size: 17px;
        color: $text-primary;
        line-height: 1.3;
        margin-top: 4px;
        padding-bottom: 10px;
        border-bottom: 1px solid $border-color;
        width: 100%;
      }

      .achievement-desc {
        font-size: 14px;
        color: $text-secondary;
        line-height: 1.5;
        padding: 0 8px;
        opacity: 0.85;
        flex-grow: 1;
      }
    }

    .confirm-section {
      position: sticky;
      bottom: 24px;
    }

    .confirm-card {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 20px;
      padding: 32px;
      background: $bg-card;
      border: 1px solid $accent-primary;
      border-radius: $radius-xl;
      box-shadow: $shadow-lg, $shadow-glow;
    }

    .confirm-preview {
      display: flex;
      align-items: center;
      gap: 16px;

      .arrow {
        font-size: 24px;
        color: $accent-primary;
      }
    }

    .confirm-text {
      font-size: 16px;
      text-align: center;

      strong {
        color: $accent-primary;
      }
    }

    .points-selector {
      display: flex;
      align-items: center;
      gap: 16px;
      width: 100%;
      max-width: 300px;

      .points-label {
        font-weight: 600;
        color: $text-secondary;
        white-space: nowrap;
      }

      .slider-container {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 8px;
      }

      .points-slider {
        width: 100%;
        height: 8px;
        border-radius: 4px;
        background: $bg-tertiary;
        outline: none;
        -webkit-appearance: none;
        appearance: none;
        cursor: pointer;

        &::-webkit-slider-thumb {
          -webkit-appearance: none;
          appearance: none;
          width: 24px;
          height: 24px;
          border-radius: 50%;
          background: $gradient-primary;
          cursor: pointer;
          box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
          transition: transform 0.15s ease;
        }

        &::-moz-range-thumb {
          width: 24px;
          height: 24px;
          border-radius: 50%;
          background: $gradient-primary;
          cursor: pointer;
          border: none;
          box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
        }

        &:hover::-webkit-slider-thumb {
          transform: scale(1.1);
        }

        &.disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      }

      .points-markers {
        display: flex;
        justify-content: space-between;
        padding: 9px;

        span {
          font-size: 12px;
          color: $text-muted;
          font-weight: 500;
          transition: all 0.2s ease;

          &.active {
            color: $accent-primary;
            font-weight: 700;
          }

          &:not(.affordable) {
            color: $text-muted;
            opacity: 0.5;
          }
        }
      }

      .points-value {
        font-weight: 700;
        color: $accent-primary;
        white-space: nowrap;
        min-width: 70px;
        text-align: right;
      }
    }

    .confirm-cost {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 16px;
      background: $bg-tertiary;
      border-radius: $radius-md;
      font-size: 14px;

      .cost-value {
        font-weight: 600;
        color: $accent-primary;
      }
    }

    .secret-toggle {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 8px;
      padding: 16px;
      border-radius: $radius-md;
      width: 100%;
      max-width: 300px;

      .toggle-container {
        display: flex;
        align-items: center;
        gap: 12px;
        cursor: pointer;
        user-select: none;
      }

      .toggle-switch {
        position: relative;
        width: 48px;
        height: 26px;
        background: $bg-card;
        border: 2px solid $border-color;
        border-radius: 13px;
        transition: all $transition-fast;

        .toggle-slider {
          position: absolute;
          top: 2px;
          left: 2px;
          width: 18px;
          height: 18px;
          background: $text-secondary;
          border-radius: 50%;
          transition: all $transition-fast;
        }

        &.active {
          background: rgba($accent-primary, 0.2);
          border-color: $accent-primary;

          .toggle-slider {
            left: 24px;
            background: $accent-primary;
          }
        }
      }

      .toggle-label {
        display: flex;
        align-items: center;
        gap: 8px;
        font-weight: 600;
        font-size: 14px;

        .toggle-icon {
          font-size: 18px;
        }
      }

      .toggle-hint {
        font-size: 12px;
        color: $text-muted;
        text-align: center;
      }
    }
  `]
})
export class RateComponent implements OnInit {
  private userService = inject(UserService);
  private achievementService = inject(AchievementService);
  private voteService = inject(VoteService);
  private notifications = inject(NotificationService);
  private settingsService = inject(SettingsService);
  private soundService = inject(SoundService);
  auth = inject(AuthService);

  users = signal<User[]>([]);
  positiveAchievements = signal<Achievement[]>([]);
  negativeAchievements = signal<Achievement[]>([]);

  // Expose votingPaused from SettingsService
  votingPaused = this.settingsService.votingPaused;

  loadingUsers = signal(true);
  submitting = signal(false);

  selectedUser = signal<User | null>(null);
  selectedAchievement = signal<Achievement | null>(null);
  selectedPoints = signal(1);
  isSecretVote = signal(false); // Will be set based on achievement type

  ngOnInit(): void {
    this.loadUsers();
    this.loadAchievements();
  }

  loadUsers(): void {
    this.userService.getOthers().subscribe({
      next: (users) => {
        this.users.set(users);
        this.loadingUsers.set(false);
      },
      error: (error) => {
        console.error('Failed to load users:', error);
        this.loadingUsers.set(false);
      }
    });
  }

  loadAchievements(): void {
    this.achievementService.getAll().subscribe({
      next: (response) => {
        this.positiveAchievements.set(response.positive || []);
        this.negativeAchievements.set(response.negative || []);
      },
      error: (error) => {
        console.error('Failed to load achievements:', error);
      }
    });
  }

  selectUser(user: User): void {
    this.selectedUser.set(user);
    this.selectedAchievement.set(null);
    this.selectedPoints.set(1);
  }

  selectAchievement(achievement: Achievement): void {
    this.selectedAchievement.set(achievement);
    this.selectedPoints.set(1);
    // Default: negative achievements are secret, positive are open
    this.isSecretVote.set(!achievement.is_positive);
  }

  toggleSecret(): void {
    this.isSecretVote.update(v => !v);
  }

  submitVote(): void {
    const user = this.selectedUser();
    const achievement = this.selectedAchievement();
    const points = this.selectedPoints();
    const isSecret = this.isSecretVote();

    if (!user || !achievement) return;

    this.submitting.set(true);

    this.voteService.create({
      to_user_id: user.id,
      achievement_id: achievement.id,
      points: points,
      is_secret: isSecret
    }).subscribe({
      next: (response) => {
        this.soundService.playReviewGiven();
        const pointsText = points === 1 ? '1 Punkt' : `${points} Punkte`;
        const secretText = isSecret ? ' (geheim)' : '';
        this.notifications.success(
          'Bewertung abgegeben!',
          `Du hast ${user.username} ${pointsText} für "${achievement.name}" gegeben.${secretText}`
        );
        this.auth.updateCredits(response.credits);
        this.selectedUser.set(null);
        this.selectedAchievement.set(null);
        this.selectedPoints.set(1);
        this.isSecretVote.set(false);
        this.submitting.set(false);
      },
      error: (error) => {
        console.error('Failed to submit vote:', error);
        this.notifications.error(
          'Fehler',
          error.error?.error || 'Vote konnte nicht gesendet werden.'
        );
        this.submitting.set(false);
      }
    });
  }
}
