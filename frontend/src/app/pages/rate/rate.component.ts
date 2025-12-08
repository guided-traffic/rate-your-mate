import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { UserService } from '../../services/user.service';
import { AchievementService } from '../../services/achievement.service';
import { VoteService } from '../../services/vote.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { SettingsService } from '../../services/settings.service';
import { User } from '../../models/user.model';
import { Achievement } from '../../models/achievement.model';

@Component({
  selector: 'app-rate',
  standalone: true,
  imports: [CommonModule],
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
                (click)="selectUser(user)"
              >
                <img
                  [src]="user.avatar_url || user.avatar_small || '/assets/default-avatar.png'"
                  [alt]="user.username"
                  class="avatar avatar-lg player-avatar"
                />
                <span class="player-name">{{ user.username }}</span>
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
                    <img [src]="achievement.image_url" [alt]="achievement.name" class="achievement-icon positive" />
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
                    <img [src]="achievement.image_url" [alt]="achievement.name" class="achievement-icon negative" />
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

            <div class="confirm-cost">
              <span>Kosten:</span>
              <span class="cost-value">💎 1 Credit</span>
            </div>

            <button
              class="btn btn-primary btn-lg"
              [disabled]="submitting() || auth.credits() < 1 || votingPaused()"
              (click)="submitVote()"
            >
              @if (submitting()) {
                <div class="spinner"></div>
                Wird gesendet...
              } @else if (votingPaused()) {
                Voting pausiert
              } @else if (auth.credits() < 1) {
                Nicht genug Credits
              } @else {
                Achievement vergeben
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
      grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
      gap: 12px;
    }

    .achievement-btn {
      display: flex;
      flex-direction: column;
      align-items: flex-start;
      gap: 8px;
      padding: 16px;
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-md;
      cursor: pointer;
      text-align: left;
      transition: all $transition-fast;

      .achievement-icon {
        width: 48px;
        height: 48px;
        border-radius: $radius-sm;
        padding: 8px;

        &.positive {
          background: rgba($accent-positive, 0.15);
          filter: invert(65%) sepia(52%) saturate(5765%) hue-rotate(103deg) brightness(96%) contrast(85%);
        }

        &.negative {
          background: rgba($accent-negative, 0.15);
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
      }

      .achievement-name {
        font-weight: 600;
        color: $text-primary;
      }

      .achievement-desc {
        font-size: 12px;
        color: $text-muted;
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
  `]
})
export class RateComponent implements OnInit {
  private userService = inject(UserService);
  private achievementService = inject(AchievementService);
  private voteService = inject(VoteService);
  private notifications = inject(NotificationService);
  private settingsService = inject(SettingsService);
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
  }

  selectAchievement(achievement: Achievement): void {
    this.selectedAchievement.set(achievement);
  }

  submitVote(): void {
    const user = this.selectedUser();
    const achievement = this.selectedAchievement();

    if (!user || !achievement) return;

    this.submitting.set(true);

    this.voteService.create({
      to_user_id: user.id,
      achievement_id: achievement.id
    }).subscribe({
      next: (response) => {
        this.notifications.success(
          'Achievement vergeben!',
          `Du hast ${user.username} als "${achievement.name}" bewertet.`
        );
        this.auth.updateCredits(response.credits);
        this.selectedUser.set(null);
        this.selectedAchievement.set(null);
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
