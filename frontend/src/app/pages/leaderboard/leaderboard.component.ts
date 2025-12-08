import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { VoteService } from '../../services/vote.service';
import { AchievementLeaderboard } from '../../models/vote.model';

@Component({
  selector: 'app-leaderboard',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="page fade-in">
      <div class="page-header">
        <h1 class="page-title">Leaderboard</h1>
        <p class="page-subtitle">Top 3 Spieler pro Achievement</p>
      </div>

      @if (loading()) {
        <div class="loading-container">
          <div class="spinner"></div>
        </div>
      } @else {
        <!-- Positive Achievements -->
        <section class="section">
          <h2 class="section-title positive">
            <span>👍</span> Positive Achievements
          </h2>

          <div class="leaderboard-grid">
            @for (item of positiveAchievements(); track item.achievement.id) {
              <div class="leaderboard-card positive">
                <div class="card-header">
                  @if (item.achievement.image_url) {
                    <img [src]="item.achievement.image_url" [alt]="item.achievement.name" class="achievement-icon" />
                  }
                  <div class="card-header-text">
                    <h3 class="achievement-name">{{ item.achievement.name }}</h3>
                    <p class="achievement-desc">{{ item.achievement.description }}</p>
                  </div>
                </div>

                <div class="leaders">
                  @if (item.leaders.length === 0) {
                    <div class="no-leaders">
                      <span class="no-leaders-icon">🏆</span>
                      <span>Noch keine Votes</span>
                    </div>
                  } @else {
                    @for (leader of item.leaders; track leader.user.id; let i = $index) {
                      <div class="leader" [class]="'rank-' + (i + 1)">
                        <span class="medal">{{ getMedal(i) }}</span>
                        <img
                          [src]="leader.user.avatar_small || '/assets/default-avatar.png'"
                          [alt]="leader.user.username"
                          class="avatar"
                        />
                        <span class="leader-name">{{ leader.user.username }}</span>
                        <span class="vote-count">{{ leader.vote_count }}×</span>
                      </div>
                    }
                  }
                </div>
              </div>
            }
          </div>
        </section>

        <!-- Negative Achievements -->
        <section class="section">
          <h2 class="section-title negative">
            <span>👎</span> Negative Achievements
          </h2>

          <div class="leaderboard-grid">
            @for (item of negativeAchievements(); track item.achievement.id) {
              <div class="leaderboard-card negative">
                <div class="card-header">
                  @if (item.achievement.image_url) {
                    <img [src]="item.achievement.image_url" [alt]="item.achievement.name" class="achievement-icon" />
                  }
                  <div class="card-header-text">
                    <h3 class="achievement-name">{{ item.achievement.name }}</h3>
                    <p class="achievement-desc">{{ item.achievement.description }}</p>
                  </div>
                </div>

                <div class="leaders">
                  @if (item.leaders.length === 0) {
                    <div class="no-leaders">
                      <span class="no-leaders-icon">🏆</span>
                      <span>Noch keine Votes</span>
                    </div>
                  } @else {
                    @for (leader of item.leaders; track leader.user.id; let i = $index) {
                      <div class="leader" [class]="'rank-' + (i + 1)">
                        <span class="medal">{{ getMedal(i) }}</span>
                        <img
                          [src]="leader.user.avatar_small || '/assets/default-avatar.png'"
                          [alt]="leader.user.username"
                          class="avatar"
                        />
                        <span class="leader-name">{{ leader.user.username }}</span>
                        <span class="vote-count">{{ leader.vote_count }}×</span>
                      </div>
                    }
                  }
                </div>
              </div>
            }
          </div>
        </section>
      }
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .loading-container {
      display: flex;
      justify-content: center;
      padding: 48px;
    }

    .section {
      margin-bottom: 48px;
    }

    .section-title {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 20px;
      margin-bottom: 24px;

      &.positive {
        color: $accent-positive;
      }

      &.negative {
        color: $accent-negative;
      }
    }

    .leaderboard-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 20px;
    }

    .leaderboard-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      overflow: hidden;

      &.positive {
        border-top: 3px solid $accent-positive;
      }

      &.negative {
        border-top: 3px solid $accent-negative;
      }
    }

    .card-header {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 16px 20px;
      background: $bg-tertiary;
      border-bottom: 1px solid $border-color;

      .achievement-icon {
        width: 48px;
        height: 48px;
        padding: 8px;
        border-radius: $radius-sm;
        flex-shrink: 0;
      }

      .card-header-text {
        flex: 1;
        min-width: 0;
      }
    }

    .positive .card-header .achievement-icon {
      background: rgba($accent-positive, 0.15);
      filter: invert(65%) sepia(52%) saturate(5765%) hue-rotate(103deg) brightness(96%) contrast(85%);
    }

    .negative .card-header .achievement-icon {
      background: rgba($accent-negative, 0.15);
      filter: invert(39%) sepia(95%) saturate(1834%) hue-rotate(336deg) brightness(96%) contrast(93%);
    }

    .achievement-name {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 4px;
    }

    .achievement-desc {
      font-size: 12px;
      color: $text-muted;
      margin: 0;
    }

    .leaders {
      padding: 16px 20px;
      display: flex;
      flex-direction: column;
      gap: 12px;
      min-height: 140px;
    }

    .no-leaders {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 8px;
      flex: 1;
      color: $text-muted;
      font-size: 14px;

      .no-leaders-icon {
        font-size: 24px;
        opacity: 0.5;
      }
    }

    .leader {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 8px 12px;
      background: $bg-tertiary;
      border-radius: $radius-md;
      transition: all $transition-fast;

      &:hover {
        background: $bg-hover;
      }

      &.rank-1 {
        background: rgba($accent-warning, 0.1);
        border: 1px solid rgba($accent-warning, 0.3);

        .medal {
          font-size: 20px;
        }
      }

      &.rank-2 {
        .medal {
          font-size: 18px;
        }
      }

      &.rank-3 {
        .medal {
          font-size: 16px;
        }
      }
    }

    .medal {
      flex-shrink: 0;
      width: 28px;
      text-align: center;
    }

    .leader-name {
      flex: 1;
      font-weight: 500;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .vote-count {
      flex-shrink: 0;
      font-weight: 600;
      color: $accent-primary;
      font-size: 14px;
    }
  `]
})
export class LeaderboardComponent implements OnInit {
  private voteService = inject(VoteService);

  positiveAchievements = signal<AchievementLeaderboard[]>([]);
  negativeAchievements = signal<AchievementLeaderboard[]>([]);
  loading = signal(true);

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.voteService.getLeaderboard().subscribe({
      next: (leaderboard) => {
        // Separate positive and negative achievements
        const positives = leaderboard.filter(lb => lb.achievement.is_positive);
        const negatives = leaderboard.filter(lb => !lb.achievement.is_positive);

        this.positiveAchievements.set(positives);
        this.negativeAchievements.set(negatives);
        this.loading.set(false);
      },
      error: (error) => {
        console.error('Failed to load leaderboard data:', error);
        this.loading.set(false);
      }
    });
  }

  getMedal(index: number): string {
    const medals = ['🥇', '🥈', '🥉'];
    return medals[index] || '';
  }
}
