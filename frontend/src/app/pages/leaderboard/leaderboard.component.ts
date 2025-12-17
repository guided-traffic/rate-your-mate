import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { VoteService } from '../../services/vote.service';
import { WebSocketService } from '../../services/websocket.service';
import { SettingsService } from '../../services/settings.service';
import { AchievementLeaderboard, ChampionsResult } from '../../models/vote.model';
import { Subscription } from 'rxjs';

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

      <!-- Countdown Section -->
      @if (hasCountdown() && !isCountdownExpired()) {
        <div class="countdown-card">
          <div class="countdown-content">
            <span class="countdown-icon">⏰</span>
            <span class="countdown-label">Countdown bis zum Start:</span>
            <div class="countdown-timer">
              <span class="countdown-value">{{ countdownDays() }}</span><span class="countdown-unit">T</span>
              <span class="countdown-separator">:</span>
              <span class="countdown-value">{{ countdownHours() }}</span><span class="countdown-unit">H</span>
              <span class="countdown-separator">:</span>
              <span class="countdown-value">{{ countdownMinutes() }}</span><span class="countdown-unit">M</span>
              <span class="countdown-separator">:</span>
              <span class="countdown-value">{{ countdownSeconds() }}</span><span class="countdown-unit">S</span>
            </div>
          </div>
        </div>
      }

      @if (loading()) {
        <div class="loading-container">
          <div class="spinner"></div>
        </div>
      } @else {
        <!-- Champions Section -->
        <section class="champions-section">
          <!-- König (1. Platz) -->
          <div class="king-wrapper">
            <div class="champion-card king">
              <div class="champion-image-wrapper">
                <img src="/logos/king.webp" alt="König" class="champion-image" />
              </div>
              <h2 class="champion-title">👑 König der LAN-Party</h2>
              <p class="champion-subtitle">1. Platz</p>
              @if (champions()?.king) {
                <div class="champion-holder">
                  <img
                    [src]="champions()!.king!.user.avatar_url || '/assets/default-avatar.png'"
                    [alt]="champions()!.king!.user.username"
                    class="champion-avatar"
                  />
                  <span class="champion-name">{{ champions()!.king!.user.username }}</span>
                  <div class="champion-stats">
                    <span class="stat total-score">{{ champions()!.king!.total_score }} Punkte</span>
                  </div>
                  <div class="champion-stats-detail">
                    <span class="stat-detail">{{ champions()!.king!.net_votes }} Votes</span>
                    @if (champions()!.king!.bonus_points > 0) {
                      <span class="stat-separator">+</span>
                      <span class="stat-detail bonus">{{ champions()!.king!.bonus_points }} Bonus</span>
                    }
                  </div>
                </div>
              } @else {
                <div class="no-champion">
                  <span>Noch nicht ermittelt</span>
                </div>
              }
            </div>
          </div>

          <!-- 2. und 3. Platz -->
          <div class="runners-up-grid">
            <!-- 2. Platz -->
            <div class="champion-card runner-up second">
              <h2 class="champion-title">🥈 2. Platz</h2>
              @if (champions()?.second) {
                <div class="champion-holder">
                  <img
                    [src]="champions()!.second!.user.avatar_url || '/assets/default-avatar.png'"
                    [alt]="champions()!.second!.user.username"
                    class="champion-avatar"
                  />
                  <span class="champion-name">{{ champions()!.second!.user.username }}</span>
                  <div class="champion-stats">
                    <span class="stat total-score">{{ champions()!.second!.total_score }} Punkte</span>
                  </div>
                  <div class="champion-stats-detail">
                    <span class="stat-detail">{{ champions()!.second!.net_votes }} Votes</span>
                    @if (champions()!.second!.bonus_points > 0) {
                      <span class="stat-separator">+</span>
                      <span class="stat-detail bonus">{{ champions()!.second!.bonus_points }} Bonus</span>
                    }
                  </div>
                </div>
              } @else {
                <div class="no-champion">
                  <span>Noch nicht ermittelt</span>
                </div>
              }
            </div>

            <!-- 3. Platz -->
            <div class="champion-card runner-up third">
              <h2 class="champion-title">🥉 3. Platz</h2>
              @if (champions()?.third) {
                <div class="champion-holder">
                  <img
                    [src]="champions()!.third!.user.avatar_url || '/assets/default-avatar.png'"
                    [alt]="champions()!.third!.user.username"
                    class="champion-avatar"
                  />
                  <span class="champion-name">{{ champions()!.third!.user.username }}</span>
                  <div class="champion-stats">
                    <span class="stat total-score">{{ champions()!.third!.total_score }} Punkte</span>
                  </div>
                  <div class="champion-stats-detail">
                    <span class="stat-detail">{{ champions()!.third!.net_votes }} Votes</span>
                    @if (champions()!.third!.bonus_points > 0) {
                      <span class="stat-separator">+</span>
                      <span class="stat-detail bonus">{{ champions()!.third!.bonus_points }} Bonus</span>
                    }
                  </div>
                </div>
              } @else {
                <div class="no-champion">
                  <span>Noch nicht ermittelt</span>
                </div>
              }
            </div>
          </div>
        </section>

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
                    <div class="achievement-icon-wrapper gold-dust">
                      <img [src]="item.achievement.image_url" [alt]="item.achievement.name" class="achievement-icon" />
                    </div>
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
                    <div class="achievement-icon-wrapper shatter-dust">
                      <img [src]="item.achievement.image_url" [alt]="item.achievement.name" class="achievement-icon" />
                    </div>
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

    .countdown-card {
      background: linear-gradient(135deg, rgba($accent-primary, 0.15), rgba($accent-secondary, 0.15));
      border: 1px solid rgba($accent-primary, 0.3);
      border-radius: $radius-lg;
      padding: 16px 24px;
      margin-bottom: 24px;

      &.expired {
        background: linear-gradient(135deg, rgba($accent-success, 0.15), rgba($accent-primary, 0.15));
        border-color: rgba($accent-success, 0.3);
      }
    }

    .countdown-content {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 12px;
      flex-wrap: wrap;
    }

    .countdown-icon {
      font-size: 24px;
    }

    .countdown-label {
      font-size: 16px;
      font-weight: 500;
      color: $text-secondary;
    }

    .countdown-text {
      font-size: 18px;
      font-weight: 600;
      color: $accent-success;
    }

    .countdown-timer {
      display: flex;
      align-items: baseline;
      gap: 2px;
      font-family: 'JetBrains Mono', monospace;
    }

    .countdown-value {
      font-size: 24px;
      font-weight: 700;
      background: $gradient-primary;
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }

    .countdown-unit {
      font-size: 12px;
      color: $text-muted;
      margin-right: 4px;
    }

    .countdown-separator {
      font-size: 20px;
      color: $text-muted;
      margin: 0 2px;
    }

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

      .achievement-icon-wrapper {
        flex-shrink: 0;
      }

      .achievement-icon {
        width: 48px;
        height: 48px;
        padding: 8px;
        border-radius: $radius-sm;
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

    // Champions Section Styles
    .champions-section {
      margin-bottom: 48px;
    }

    .king-wrapper {
      display: flex;
      justify-content: center;
      margin-bottom: 24px;
    }

    .runners-up-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 20px;
      max-width: 600px;
      margin: 0 auto;
    }

    .champion-card {
      background: $bg-card;
      border: 2px solid $border-color;
      border-radius: $radius-lg;
      padding: 24px;
      text-align: center;
      transition: all $transition-fast;

      &.king {
        border-color: rgba(255, 215, 0, 0.5);
        background: linear-gradient(135deg, rgba(255, 215, 0, 0.15) 0%, $bg-card 100%);
        max-width: 400px;

        .champion-title {
          color: #ffd700;
          font-size: 28px;
        }

        .champion-avatar {
          width: 80px;
          height: 80px;
          border-color: #ffd700;
        }

        .champion-name {
          font-size: 22px;
        }

        .total-score {
          font-size: 18px;
          color: #ffd700;
        }
      }

      &.runner-up {
        padding: 20px;

        .champion-title {
          font-size: 18px;
          margin-bottom: 12px;
        }

        .champion-avatar {
          width: 56px;
          height: 56px;
        }

        .champion-name {
          font-size: 16px;
        }

        .champion-holder {
          padding: 12px;
        }
      }

      &.second {
        border-color: rgba(192, 192, 192, 0.5);
        background: linear-gradient(135deg, rgba(192, 192, 192, 0.1) 0%, $bg-card 100%);

        .champion-title {
          color: #c0c0c0;
        }

        .champion-avatar {
          border-color: #c0c0c0;
        }

        .total-score {
          color: #c0c0c0;
        }
      }

      &.third {
        border-color: rgba(205, 127, 50, 0.5);
        background: linear-gradient(135deg, rgba(205, 127, 50, 0.1) 0%, $bg-card 100%);

        .champion-title {
          color: #cd7f32;
        }

        .champion-avatar {
          border-color: #cd7f32;
        }

        .total-score {
          color: #cd7f32;
        }
      }
    }

    .champion-image-wrapper {
      margin-bottom: 16px;
    }

    .champion-image {
      width: 120px;
      height: 120px;
      object-fit: contain;
      border-radius: $radius-md;
    }

    .champion-title {
      font-size: 24px;
      font-weight: 700;
      margin-bottom: 4px;
    }

    .champion-subtitle {
      font-size: 14px;
      color: $text-muted;
      margin-bottom: 20px;
    }

    .champion-holder {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 8px;
      padding: 16px;
      background: $bg-tertiary;
      border-radius: $radius-md;
    }

    .champion-avatar {
      width: 64px;
      height: 64px;
      border-radius: 50%;
      border: 3px solid $border-color;
    }

    .champion-name {
      font-size: 18px;
      font-weight: 600;
    }

    .champion-stats {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 15px;
      font-weight: 600;

      .total-score {
        font-weight: 700;
      }
    }

    .champion-stats-detail {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 12px;
      color: $text-muted;

      .stat-detail {
        &.bonus {
          color: $accent-positive;
        }
      }
    }

    .stat-separator {
      opacity: 0.5;
    }

    .no-champion {
      padding: 24px;
      color: $text-muted;
      font-style: italic;
    }
  `]
})
export class LeaderboardComponent implements OnInit, OnDestroy {
  private voteService = inject(VoteService);
  private wsService = inject(WebSocketService);
  private settingsService = inject(SettingsService);

  positiveAchievements = signal<AchievementLeaderboard[]>([]);
  negativeAchievements = signal<AchievementLeaderboard[]>([]);
  champions = signal<ChampionsResult | null>(null);
  loading = signal(true);

  // Countdown
  private countdownInterval: ReturnType<typeof setInterval> | null = null;
  private countdownTargetTime = signal<Date | null>(null);
  private currentTime = signal<Date>(new Date());

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

  private newVoteSubscription?: Subscription;
  private voteInvalidationSubscription?: Subscription;
  private settingsSubscription?: Subscription;

  ngOnInit(): void {
    this.loadData();
    this.subscribeToVoteUpdates();
    this.initCountdown();
  }

  ngOnDestroy(): void {
    this.newVoteSubscription?.unsubscribe();
    this.voteInvalidationSubscription?.unsubscribe();
    this.settingsSubscription?.unsubscribe();
    if (this.countdownInterval) {
      clearInterval(this.countdownInterval);
    }
  }

  private initCountdown(): void {
    // Load initial countdown from settings service
    const countdownTarget = this.settingsService.countdownTarget();
    if (countdownTarget) {
      this.countdownTargetTime.set(new Date(countdownTarget));
    }

    // Subscribe to settings updates (via WebSocket)
    this.settingsSubscription = this.wsService.settingsUpdate$.subscribe((settings) => {
      if (settings.countdown_target !== undefined) {
        if (settings.countdown_target) {
          this.countdownTargetTime.set(new Date(settings.countdown_target));
        } else {
          this.countdownTargetTime.set(null);
        }
      }
    });

    // Update current time every second
    this.countdownInterval = setInterval(() => {
      this.currentTime.set(new Date());
    }, 1000);
  }

  private subscribeToVoteUpdates(): void {
    // Reload data when a new vote is created
    this.newVoteSubscription = this.wsService.newVote$.subscribe(() => {
      console.log('Leaderboard: New vote received, reloading data');
      this.loadData();
    });

    // Reload data when a vote is invalidated/revalidated
    this.voteInvalidationSubscription = this.wsService.voteInvalidation$.subscribe(() => {
      console.log('Leaderboard: Vote invalidation changed, reloading data');
      this.loadData();
    });
  }

  loadData(): void {
    // Load leaderboard data
    this.voteService.getLeaderboard().subscribe({
      next: (leaderboard) => {
        const positives = leaderboard.filter(lb => lb.achievement.is_positive);
        const negatives = leaderboard.filter(lb => !lb.achievement.is_positive);

        this.positiveAchievements.set(positives);
        this.negativeAchievements.set(negatives);
      },
      error: (error) => {
        console.error('Failed to load leaderboard data:', error);
      }
    });

    // Load champions data
    this.voteService.getChampions().subscribe({
      next: (champions) => {
        this.champions.set(champions);
        this.loading.set(false);
      },
      error: (error) => {
        console.error('Failed to load champions data:', error);
        this.loading.set(false);
      }
    });
  }

  getMedal(index: number): string {
    const medals = ['🥇', '🥈', '🥉'];
    return medals[index] || '';
  }
}
