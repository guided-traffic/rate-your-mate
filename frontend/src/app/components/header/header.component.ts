import { Component, inject, OnInit, OnDestroy, signal, computed, effect, HostListener, ElementRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { WebSocketService } from '../../services/websocket.service';
import { NotificationService } from '../../services/notification.service';
import { SettingsService } from '../../services/settings.service';
import { SoundService } from '../../services/sound.service';
import { RankingService, PlayerRanking, GlobalRankingResponse } from '../../services/ranking.service';
import { UserService } from '../../services/user.service';
import { User } from '../../models/user.model';
import { Subscription, interval } from 'rxjs';

@Component({
  selector: 'app-header',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive],
  template: `
    <header class="header">
      <div class="header-content">
        <div class="header-left">
          <a routerLink="/" class="logo">
            <span class="logo-icon">🎮</span>
            <span class="logo-text">Rate your Mate</span>
          </a>

          @if (auth.isAuthenticated()) {
            <nav class="nav">
              <a routerLink="/games" routerLinkActive="active" class="nav-link">
                <span class="nav-icon">🎮</span>
                Games
              </a>
              <a routerLink="/rate" routerLinkActive="active" class="nav-link">
                <span class="nav-icon">⭐</span>
                Rate Player
              </a>
              <a routerLink="/leaderboard" routerLinkActive="active" class="nav-link">
                <span class="nav-icon">🏆</span>
                Leaderboard
              </a>
              <a routerLink="/timeline" routerLinkActive="active" class="nav-link">
                <span class="nav-icon">📜</span>
                Timeline
              </a>
            </nav>
          }
        </div>

        <div class="header-right">
          @if (auth.isAuthenticated()) {
            <div class="credits-bar-container" [title]="auth.credits() + ' / ' + maxCredits() + ' Credits'">
              <span class="credits-icon">💎</span>
              <div class="credits-bar">
                @for (i of creditSlots(); track i) {
                  <div
                    class="credit-slot"
                    [class.filled]="i < auth.credits()"
                    [class.charging]="i === auth.credits() && auth.credits() < maxCredits()"
                  >
                    @if (i === auth.credits() && auth.credits() < maxCredits()) {
                      <div
                        class="charge-progress"
                        [style.height.%]="chargeProgress()"
                      ></div>
                    }
                  </div>
                }
              </div>
              <span class="credits-countdown" [class.full]="auth.credits() >= maxCredits()">{{ countdownDisplay() }}</span>
              <span class="credits-count">{{ auth.credits() }}</span>
            </div>

            <div class="user-menu" (click)="toggleMenu()">
              <img
                [src]="auth.user()?.avatar_small || auth.user()?.avatar_url || '/assets/default-avatar.png'"
                [alt]="auth.user()?.username"
                class="avatar"
              />
              <span class="username">{{ auth.user()?.username }}</span>
              <span class="dropdown-arrow">▼</span>

              @if (menuOpen) {
                <div class="dropdown-menu">
                  <div class="dropdown-header">
                    <span class="dropdown-steam-id">ID: {{ auth.user()?.steam_id }}</span>
                    <button class="copy-btn" (click)="copySteamId($event)" [title]="copied() ? 'Kopiert!' : 'ID kopieren'">
                      {{ copied() ? '✓' : '📋' }}
                    </button>
                  </div>
                  <a [href]="auth.user()?.profile_url" target="_blank" class="dropdown-item">
                    <span>🔗</span> Steam Profile
                  </a>
                  @if (isAdmin()) {
                    <a routerLink="/admin" class="dropdown-item" (click)="navigateToAdmin($event)">
                      <span>⚙️</span> Admin
                    </a>
                  }
                  <button (click)="logout()" class="dropdown-item logout">
                    <span>🚪</span> Logout
                  </button>
                </div>
              }
            </div>

            <div
              class="rank-badge-container"
              (click)="toggleRankingOverlay($event)"
            >
              <div class="rank-badge" [title]="rankingService.isRankingActive() ? 'Dein aktueller Platz' : 'Noch ' + rankingService.votesUntilRanking() + ' Votes bis zum Ranking'">
                @if (rankingService.rankDisplay()) {
                  <span class="rank-number">{{ rankingService.rankDisplay() }}</span>
                } @else {
                  <span class="rank-hourglass">⏳</span>
                }
              </div>

              @if (rankingOverlayVisible()) {
                <div class="ranking-overlay" (click)="$event.stopPropagation()">
                  <div class="ranking-overlay-header">
                    <span class="ranking-overlay-title">🏆 Spieler-Ranking</span>
                  </div>
                  <div class="ranking-chart">
                    <div class="players-track">
                      @for (player of sortedPlayers(); track player.user.id; let i = $index) {
                        <div
                          class="player-marker"
                          [style.left.%]="getPlayerPosition(player.net_votes)"
                          [style.top.%]="getPlayerYPosition(i)"
                          [style.z-index]="getZIndex(player.net_votes)"
                          [title]="player.user.username + ': ' + player.net_votes + ' Punkte'"
                        >
                          <img
                            [src]="player.user.avatar_small || player.user.avatar_url || '/assets/default-avatar.png'"
                            [alt]="player.user.username"
                            class="player-avatar"
                            [class]="'player-avatar wobble-' + (i % 6)"
                          />
                        </div>
                      }
                    </div>
                    <div class="x-axis">
                      @for (tick of axisTicks(); track tick) {
                        <div class="tick" [style.left.%]="getTickPosition(tick)" [class.zero-tick]="tick === 0">
                          <span class="tick-label">{{ tick }}</span>
                        </div>
                      }
                    </div>
                  </div>
                </div>
              }
            </div>

          }
        </div>
      </div>
    </header>
  `,
  styles: [`
    @use 'variables' as *;

    .header {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      height: 64px;
      background: $bg-secondary;
      border-bottom: 1px solid $border-color;
      z-index: 100;
    }

    .header-content {
      max-width: 1250px;
      margin: 0 auto;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 24px;
      gap: 14px;
    }

    .header-left {
      display: flex;
      align-items: center;
      gap: 22px;
      flex: 1;
    }

    .logo {
      display: flex;
      align-items: center;
      gap: 10px;
      text-decoration: none;
      color: $text-primary;

      .logo-icon {
        font-size: 24px;
      }

      .logo-text {
        font-size: 20px;
        font-weight: 700;
        background: $gradient-primary;
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
      }
    }

    .nav {
      display: flex;
      gap: 8px;
    }

    .nav-link {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 8px 16px;
      border-radius: $radius-md;
      color: $text-secondary;
      text-decoration: none;
      font-size: 14px;
      font-weight: 500;
      transition: all $transition-fast;

      &:hover {
        background: $bg-tertiary;
        color: $text-primary;
      }

      &.active {
        background: rgba($accent-primary, 0.15);
        color: $accent-primary;
      }

      &.admin-link {
        border-left: 1px solid $border-color;
        margin-left: 8px;
        padding-left: 16px;
      }

      .nav-icon {
        font-size: 16px;
      }
    }

    .header-right {
      display: flex;
      align-items: center;
      gap: 16px;
    }

    .credits-bar-container {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 12px;
      background: $bg-tertiary;
      border: 1px solid $border-color;
      border-radius: $radius-full;

      .credits-icon {
        font-size: 14px;
      }

      .credits-bar {
        display: flex;
        gap: 2px;
      }

      .credit-slot {
        width: 12px;
        height: 16px;
        background: $bg-primary;
        border: 1px solid $border-color;
        border-radius: 2px;
        position: relative;
        overflow: hidden;
        transition: background $transition-fast;

        &.filled {
          background: $accent-primary;
          border-color: $accent-secondary;
          box-shadow: 0 0 4px rgba($accent-primary, 0.4);
        }

        &.charging {
          background: $bg-primary;

          // Schraffierter Hintergrund
          &::before {
            content: '';
            position: absolute;
            inset: 0;
            background: repeating-linear-gradient(
              -45deg,
              transparent,
              transparent 2px,
              rgba($accent-primary, 0.2) 2px,
              rgba($accent-primary, 0.2) 4px
            );
          }
        }

        .charge-progress {
          position: absolute;
          bottom: 0;
          left: 0;
          right: 0;
          background: linear-gradient(to top, $accent-primary, $accent-secondary);
          transition: height 1s linear;
        }
      }

      .credits-countdown {
        font-size: 12px;
        font-weight: 500;
        font-family: monospace;
        color: $text-muted;
        min-width: 32px;
        text-align: center;

        &.full {
          color: $text-muted;
          opacity: 0.5;
        }
      }

      .credits-count {
        font-size: 13px;
        font-weight: 600;
        color: $accent-primary;
        min-width: 16px;
        text-align: center;
      }
    }

    .user-menu {
      position: relative;
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 6px 12px 6px 6px;
      background: $bg-tertiary;
      border: 1px solid $border-color;
      border-radius: $radius-full;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover {
        border-color: $border-light;
      }

      .avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
      }

      .username {
        font-size: 14px;
        font-weight: 500;
      }

      .dropdown-arrow {
        font-size: 10px;
        color: $text-muted;
      }
    }

    .dropdown-menu {
      position: absolute;
      top: calc(100% + 8px);
      right: 0;
      min-width: 180px;
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-md;
      box-shadow: $shadow-lg;
      overflow: hidden;
      animation: fadeIn 0.15s ease;
    }

    .dropdown-header {
      padding: 10px 16px;
      background: $bg-tertiary;
      border-bottom: 1px solid $border-color;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
    }

    .dropdown-steam-id {
      font-size: 12px;
      font-family: monospace;
      color: $text-muted;
    }

    .copy-btn {
      background: none;
      border: 1px solid $border-color;
      border-radius: $radius-sm;
      padding: 4px 8px;
      font-size: 12px;
      cursor: pointer;
      color: $text-muted;
      transition: all $transition-fast;

      &:hover {
        background: $bg-hover;
        color: $text-primary;
        border-color: $accent-primary;
      }
    }

    .dropdown-item {
      display: flex;
      align-items: center;
      gap: 10px;
      width: 100%;
      padding: 12px 16px;
      background: none;
      border: none;
      color: $text-primary;
      font-size: 14px;
      text-decoration: none;
      cursor: pointer;
      transition: background $transition-fast;

      &:hover {
        background: $bg-hover;
      }

      &.logout {
        color: $accent-error;
        border-top: 1px solid $border-color;
      }
    }

    .rank-badge-container {
      position: relative;
    }

    .rank-badge {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 40px;
      height: 40px;
      background: $gradient-primary;
      border: 2px solid #FFD700;
      border-radius: 50%;
      font-size: 14px;
      font-weight: 700;
      color: $text-primary;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3), 0 0 8px rgba(255, 215, 0, 0.3);
      flex-shrink: 0;
      cursor: pointer;

      .rank-number {
        line-height: 1;
      }

      .rank-hourglass {
        font-size: 16px;
        line-height: 1;
      }
    }

    .ranking-overlay {
      position: absolute;
      top: calc(100% + 12px);
      right: 0;
      width: 800px;
      max-height: 90vh;
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      box-shadow: $shadow-lg;
      overflow: hidden;
      animation: fadeIn 0.2s ease;
      z-index: 1000;
    }

    .ranking-overlay-header {
      padding: 20px 24px;
      background: $bg-tertiary;
      border-bottom: 1px solid $border-color;
    }

    .ranking-overlay-title {
      font-size: 18px;
      font-weight: 600;
      color: $text-primary;
    }

    .ranking-chart {
      padding: 20px 24px 32px;
      display: flex;
      flex-direction: column;
      gap: 10px;
    }

    .players-track {
      position: relative;
      height: 240px;
      margin-bottom: 8px;
    }

    .player-marker {
      position: absolute;
      transform: translate(-50%, -50%);
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 4px;
      transition: left 0.3s ease, top 0.3s ease;
    }

    .player-avatar {
      width: 40px;
      height: 40px;
      border-radius: 50%;
      border: 2px solid $border-color;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
      cursor: pointer;

      &:hover {
        animation: none !important;
        transform: scale(1.15);
        z-index: 1000 !important;
      }
    }

    .wobble-0 { animation: wobble0 3.0s ease-in-out infinite; }
    .wobble-1 { animation: wobble1 3.2s ease-in-out infinite; }
    .wobble-2 { animation: wobble2 2.8s ease-in-out infinite; }
    .wobble-3 { animation: wobble3 3.4s ease-in-out infinite; }
    .wobble-4 { animation: wobble4 2.9s ease-in-out infinite; }
    .wobble-5 { animation: wobble5 3.1s ease-in-out infinite; }

    @keyframes wobble0 {
      0%, 100% { transform: translate(0, 0); }
      25% { transform: translate(3px, -2px); }
      50% { transform: translate(-1px, 3px); }
      75% { transform: translate(-3px, -1px); }
    }

    @keyframes wobble1 {
      0%, 100% { transform: translate(0, 0); }
      25% { transform: translate(-2px, 3px); }
      50% { transform: translate(3px, 1px); }
      75% { transform: translate(1px, -3px); }
    }

    @keyframes wobble2 {
      0%, 100% { transform: translate(0, 0); }
      25% { transform: translate(2px, 2px); }
      50% { transform: translate(-3px, -1px); }
      75% { transform: translate(1px, 3px); }
    }

    @keyframes wobble3 {
      0%, 100% { transform: translate(0, 0); }
      25% { transform: translate(-3px, -3px); }
      50% { transform: translate(2px, -2px); }
      75% { transform: translate(-1px, 3px); }
    }

    @keyframes wobble4 {
      0%, 100% { transform: translate(0, 0); }
      25% { transform: translate(1px, -3px); }
      50% { transform: translate(-2px, 2px); }
      75% { transform: translate(3px, 1px); }
    }

    @keyframes wobble5 {
      0%, 100% { transform: translate(0, 0); }
      25% { transform: translate(-1px, 2px); }
      50% { transform: translate(3px, -3px); }
      75% { transform: translate(-2px, -1px); }
    }

    .x-axis {
      position: relative;
      height: 28px;
      border-top: 2px solid $border-color;
    }

    .tick {
      position: absolute;
      transform: translateX(-50%);
      display: flex;
      flex-direction: column;
      align-items: center;

      &::before {
        content: '';
        width: 1px;
        height: 8px;
        background: $border-color;
      }

      &.zero-tick::before {
        width: 2px;
        background: $text-secondary;
      }
    }

    .tick-label {
      font-size: 11px;
      color: $text-muted;
      margin-top: 4px;
      font-weight: 500;
    }

@keyframes fadeIn {
      from {
        opacity: 0;
        transform: translateY(-8px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
  `]
})
export class HeaderComponent implements OnInit, OnDestroy {
  auth = inject(AuthService);
  ws = inject(WebSocketService);
  rankingService = inject(RankingService);
  private userService = inject(UserService);
  private router = inject(Router);
  private notifications = inject(NotificationService);
  private settingsService = inject(SettingsService);
  private soundService = inject(SoundService);
  private subscription?: Subscription;
  private settingsSubscription?: Subscription;
  private creditsResetSubscription?: Subscription;
  private creditsGivenSubscription?: Subscription;
  private newKingSubscription?: Subscription;
  private timerSubscription?: Subscription;
  private timerInitialized = false;

  menuOpen = false;
  copied = signal(false);

  // Ranking overlay state
  rankingOverlayVisible = signal(false);
  private allPlayers = signal<PlayerRanking[]>([]);

  // Sorted players by username
  sortedPlayers = computed(() => {
    return [...this.allPlayers()].sort((a, b) =>
      a.user.username.toLowerCase().localeCompare(b.user.username.toLowerCase())
    );
  });

  // Dynamic axis calculation
  private axisMin = computed(() => {
    const players = this.allPlayers();
    if (players.length === 0) return 0;
    const minVotes = Math.min(...players.map(p => p.net_votes));
    // If someone is negative, extend axis to include them (rounded down to nearest 5)
    if (minVotes < 0) {
      return Math.floor(minVotes / 5) * 5;
    }
    return 0;
  });

  private axisMax = computed(() => {
    const players = this.allPlayers();
    if (players.length === 0) return 10;
    const maxVotes = Math.max(...players.map(p => p.net_votes), 0);
    // Start at 10, then increase in 5er steps if someone exceeds
    if (maxVotes >= 10) {
      return Math.ceil(maxVotes / 5) * 5;
    }
    return 10;
  });

  // Generate ticks for the x-axis
  axisTicks = computed(() => {
    const min = this.axisMin();
    const max = this.axisMax();
    const ticks: number[] = [];

    // Add ticks every 5 units, but always include 0
    for (let i = min; i <= max; i += 5) {
      ticks.push(i);
    }
    // Ensure 0 is included if not already
    if (!ticks.includes(0)) {
      ticks.push(0);
      ticks.sort((a, b) => a - b);
    }
    return ticks;
  });

  // Signal for tracking seconds until next credit
  private secondsUntilCredit = signal(0);

  // Signals for settings (can be overridden by WebSocket updates)
  private settingsMaxCredits = signal<number | null>(null);
  private settingsCreditIntervalSeconds = signal<number | null>(null);

  // Computed: is the current user an admin?
  isAdmin = computed(() => this.auth.user()?.is_admin ?? false);

  // Computed values from user data (with settings override)
  maxCredits = computed(() =>
    this.settingsMaxCredits() ?? this.auth.user()?.credit_max ?? 10
  );
  creditIntervalSeconds = computed(() =>
    this.settingsCreditIntervalSeconds() ?? this.auth.user()?.credit_interval_seconds ?? 600
  );

  // Array of slot indices for the template
  creditSlots = computed(() => Array.from({ length: this.maxCredits() }, (_, i) => i));

  // Progress percentage for the charging credit (0-100)
  chargeProgress = computed(() => {
    // No progress animation when voting is paused
    if (this.settingsService.votingPaused()) return 0;
    const seconds = this.secondsUntilCredit();
    const intervalSeconds = this.creditIntervalSeconds();
    if (seconds <= 0 || this.auth.credits() >= this.maxCredits()) return 0;
    const progress = ((intervalSeconds - seconds) / intervalSeconds) * 100;
    return Math.max(0, Math.min(100, progress));
  });

  // Formatted countdown display (m:ss or -:-- or ⏸ when paused)
  countdownDisplay = computed(() => {
    // When voting is paused, credit generation is also paused
    if (this.settingsService.votingPaused()) {
      return '⏸';
    }
    if (this.auth.credits() >= this.maxCredits()) {
      return '-:--';
    }
    const seconds = this.secondsUntilCredit();
    if (seconds <= 0) {
      return '0:00';
    }
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  });

  constructor() {
    // Update secondsUntilCredit when user data changes
    effect(() => {
      const user = this.auth.user();
      if (user && user.seconds_until_credit !== undefined) {
        this.secondsUntilCredit.set(user.seconds_until_credit);
      }
    });
  }

  ngOnInit(): void {
    // Initialize credit timer when authenticated
    if (this.auth.isAuthenticated()) {
      this.initCreditTimer();
      // Load initial ranking
      this.rankingService.loadMyRanking();
    }

    // Listen for vote notifications - show popup only if current user is the recipient
    this.subscription = this.ws.newVote$.subscribe((payload) => {
      const currentUser = this.auth.user();
      if (currentUser && payload.to_user_id === currentUser.id) {
        // Play sound based on whether the review is positive or negative
        if (payload.is_positive) {
          this.soundService.playGoodReview();
        } else {
          this.soundService.playBadReview();
        }
        this.notifications.voteReceived(
          payload.from_username,
          payload.achievement_name,
          payload.from_avatar,
          payload.is_positive
        );
        // Refresh user data to update any stats
        this.auth.refreshUser();
      }
      // Refresh ranking on any new vote
      this.rankingService.refresh();
    });

    // Listen for settings updates from admin
    this.settingsSubscription = this.ws.settingsUpdate$.subscribe((settings) => {
      console.log('Settings updated via WebSocket:', settings);

      // Check if voting_paused status actually changed before showing notification
      const wasVotingPaused = this.settingsService.votingPaused();
      const isNowPaused = settings.voting_paused;

      this.settingsMaxCredits.set(settings.credit_max);
      this.settingsCreditIntervalSeconds.set(settings.credit_interval_minutes * 60);
      this.settingsService.applySettingsUpdate(settings);

      // Only show notification if voting_paused status actually changed
      if (wasVotingPaused !== isNowPaused) {
        if (isNowPaused) {
          this.notifications.info('⏸️ Voting pausiert', 'Der Admin hat das Voting pausiert');
        } else {
          this.notifications.info('▶️ Voting fortgesetzt', 'Das Voting wurde wieder aktiviert');
          // Refresh user data to get current credit state after voting is resumed
          // This ensures the timer starts with the correct value from the backend
          this.auth.refreshUser();
        }
      }
    });

    // Listen for credits reset from admin
    this.creditsResetSubscription = this.ws.creditsReset$.subscribe(() => {
      console.log('Credits reset via WebSocket');
      this.auth.refreshUser();
      this.notifications.info('🔄 Credits zurückgesetzt', 'Der Admin hat alle Credits auf 0 gesetzt');
    });

    // Listen for credits given from admin
    this.creditsGivenSubscription = this.ws.creditsGiven$.subscribe(() => {
      console.log('Credits given via WebSocket');
      this.soundService.playNewCredit();
      this.auth.refreshUser();
      this.notifications.success('🎁 Credit erhalten', 'Der Admin hat dir 1 Credit gegeben');
    });

    // Listen for new king notifications
    this.newKingSubscription = this.ws.newKing$.subscribe((payload) => {
      console.log('New king via WebSocket:', payload);
      this.soundService.playNewKing();
      this.notifications.success('👑 Neuer König!', `${payload.username} ist der neue König der LAN-Party!`);
    });
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
    this.settingsSubscription?.unsubscribe();
    this.creditsResetSubscription?.unsubscribe();
    this.creditsGivenSubscription?.unsubscribe();
    this.newKingSubscription?.unsubscribe();
    this.timerSubscription?.unsubscribe();
  }

  private initCreditTimer(): void {
    // Update every second
    this.timerSubscription = interval(1000).subscribe(() => {
      // Skip credit generation when voting is paused
      if (this.settingsService.votingPaused()) {
        return;
      }

      const current = this.secondsUntilCredit();
      if (current > 0) {
        this.secondsUntilCredit.set(current - 1);
        // Mark timer as initialized after first tick with valid time
        this.timerInitialized = true;
      } else if (this.auth.credits() < this.maxCredits()) {
        // Credit should have been earned - only play sound if timer was running
        if (this.timerInitialized) {
          this.soundService.playNewCredit();
        }
        this.auth.refreshUser();
        // Reset timer with interval from config
        this.secondsUntilCredit.set(this.creditIntervalSeconds());
        this.timerInitialized = true;
      }
    });
  }

  toggleMenu(): void {
    this.menuOpen = !this.menuOpen;
  }

  closeMenu(): void {
    this.menuOpen = false;
  }

  navigateToAdmin(event: Event): void {
    event.preventDefault();
    event.stopPropagation();
    this.menuOpen = false;
    this.router.navigate(['/admin']);
  }

  logout(): void {
    this.ws.disconnect();
    this.auth.logout();
    this.menuOpen = false;
  }

  copySteamId(event: Event): void {
    event.stopPropagation();
    const steamId = this.auth.user()?.steam_id;
    if (steamId) {
      navigator.clipboard.writeText(steamId).then(() => {
        this.copied.set(true);
        setTimeout(() => this.copied.set(false), 2000);
      });
    }
  }

  // Ranking overlay methods
  toggleRankingOverlay(event: Event): void {
    event.stopPropagation();
    if (this.rankingOverlayVisible()) {
      this.rankingOverlayVisible.set(false);
    } else {
      this.rankingOverlayVisible.set(true);
      this.loadRankingData();
    }
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: Event): void {
    if (this.rankingOverlayVisible()) {
      this.rankingOverlayVisible.set(false);
    }
  }

  private loadRankingData(): void {
    this.rankingService.loadGlobalRanking().then((response: GlobalRankingResponse) => {
      this.allPlayers.set(response.rankings);
    }).catch(err => {
      console.error('Failed to load ranking data for overlay:', err);
    });
  }

  // Calculate the position of a player icon on the track (percentage)
  getPlayerPosition(netVotes: number): number {
    const min = this.axisMin();
    const max = this.axisMax();
    const range = max - min;
    if (range === 0) return 50;
    return ((netVotes - min) / range) * 100;
  }

  // Calculate the Y position of a player based on their index (sorted by name)
  getPlayerYPosition(index: number): number {
    const totalPlayers = this.sortedPlayers().length;
    if (totalPlayers <= 1) return 50;
    // Distribute players evenly, with some padding at top and bottom
    const padding = 10; // 10% padding from edges
    const availableSpace = 100 - (2 * padding);
    return padding + (index / (totalPlayers - 1)) * availableSpace;
  }

  // Calculate z-index based on score (higher score = higher z-index)
  getZIndex(netVotes: number): number {
    return 100 + netVotes;
  }

  // Calculate tick position (percentage)
  getTickPosition(tick: number): number {
    const min = this.axisMin();
    const max = this.axisMax();
    const range = max - min;
    if (range === 0) return 0;
    return ((tick - min) / range) * 100;
  }
}
