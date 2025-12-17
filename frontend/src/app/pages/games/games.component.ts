import { Component, OnInit, OnDestroy, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { GameService } from '../../services/game.service';
import { UserService } from '../../services/user.service';
import { AuthService } from '../../services/auth.service';
import { WebSocketService } from '../../services/websocket.service';
import { Game, SyncStatus } from '../../models/game.model';
import { User } from '../../models/user.model';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-games',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="games-page fade-in">
      <div class="page-header">
        <h1>
          <span class="page-icon">🎮</span>
          Multiplayer Games
        </h1>
        <p class="subtitle">Spiele die von LAN-Party Teilnehmern besessen werden</p>
        <div class="header-actions">
          <button
            class="refresh-my-games-btn"
            (click)="refreshMyGames()"
            [disabled]="refreshingMyGames() || refreshCooldownRemaining() > 0"
            [title]="getRefreshButtonTitle()">
            @if (refreshingMyGames()) {
              <span class="btn-spinner"></span>
              Aktualisiere...
            } @else if (refreshCooldownRemaining() > 0) {
              <span class="cooldown-icon">⏱️</span>
              {{ formatCooldown(refreshCooldownRemaining()) }}
            } @else {
              <span class="refresh-icon">🔄</span>
              Meine Profil von Steam aktualisieren
            }
          </button>
        </div>
      </div>

      <!-- Sync Status Banner -->
      @if (isSyncing()) {
        <div class="sync-banner">
          <div class="sync-content">
            <div class="sync-spinner"></div>
            <div class="sync-info">
              <span class="sync-text">
                @if (syncPhase() === 'fetching_users') {
                  Lade Spielerbibliotheken...
                } @else if (syncPhase() === 'fetching_categories') {
                  Aktualisiere Spieldetails...
                } @else {
                  Synchronisiere Bibliothek...
                }
              </span>
              @if (syncTotal() > 0) {
                <div class="sync-progress">
                  <div class="progress-bar">
                    <div class="progress-fill" [style.width.%]="syncPercentage()"></div>
                  </div>
                  <span class="progress-text">{{ syncProcessed() }}/{{ syncTotal() }} ({{ syncPercentage() }}%)</span>
                </div>
              }
              @if (syncCurrentGame()) {
                <span class="current-game">{{ syncCurrentGame() }}</span>
              }
            </div>
          </div>
        </div>
      }

      @if (loading()) {
        <div class="loading">
          <div class="spinner"></div>
          <p>Lade Spiele von allen Teilnehmern...</p>
        </div>
      } @else if (error()) {
        <div class="error">
          <span class="error-icon">❌</span>
          <p>{{ error() }}</p>
          <button (click)="loadGames()">Erneut versuchen</button>
        </div>
      } @else {
        <!-- Pinned Games Section -->
        @if (pinnedGames().length > 0) {
          <section class="games-section pinned-section">
            <h2>
              <span class="section-icon">📌</span>
              Pinned Games
              <span class="count">{{ pinnedGames().length }}</span>
            </h2>
            <div class="games-grid">
              @for (game of pinnedGames(); track game.app_id) {
                <div class="game-card pinned" (click)="openSteamStore(game.app_id)">
                  <div class="game-image">
                    <img [src]="game.header_image_url" [alt]="game.name" loading="lazy" />
                  </div>
                  <div class="game-info">
                    <div class="game-title-row">
                      <h3>{{ game.name }}</h3>
                      <div class="price-review-column">
                        @if (game.price_formatted) {
                          <div class="price-badge" [class.free]="game.is_free" [class.discount]="game.discount_percent > 0">
                            @if (game.discount_percent > 0) {
                              <span class="discount-tag">-{{ game.discount_percent }}%</span>
                            }
                            @if (getPriceTier(game.price_cents, game.is_free)) {
                              <span class="price-tier">{{ getPriceTier(game.price_cents, game.is_free) }}</span>
                            }
                            <span class="price">{{ game.price_formatted }}</span>
                          </div>
                        }
                        @if (game.review_score >= 0) {
                          <div class="review-score" [class.positive]="game.review_score >= 85" [class.mixed]="game.review_score >= 70 && game.review_score < 85" [class.negative]="game.review_score < 70">
                            <span class="thumb">👍</span>
                            <span class="score">{{ game.review_score }}%</span>
                          </div>
                        }
                      </div>
                    </div>
                    <div class="game-meta">
                      @if (game.owner_count > 0) {
                        <div class="owners has-tooltip">
                          <span class="owner-icon">👥</span>
                          <span>{{ game.owner_count }} {{ game.owner_count === 1 ? 'Besitzer' : 'Besitzer' }}</span>
                          <div class="owner-tooltip">
                            <div class="tooltip-title">Besitzer:</div>
                            @for (ownerName of getOwnerNameList(game.owners); track ownerName) {
                              <div class="tooltip-owner">{{ ownerName }}</div>
                            }
                          </div>
                        </div>
                      } @else {
                        <div class="owners no-owners">
                          <span class="owner-icon">👤</span>
                          <span>Kein Teilnehmer besitzt dieses Spiel</span>
                        </div>
                      }
                      <div class="categories">
                        @for (cat of getMultiplayerCategories(game.categories); track cat) {
                          <span class="category-tag">{{ cat }}</span>
                        }
                      </div>
                    </div>
                  </div>
                </div>
              }
            </div>
          </section>
        }

        <!-- All Games Section -->
        <section class="games-section">
          <h2>
            <span class="section-icon">🎲</span>
            Alle Multiplayer Spiele
            <span class="count">{{ allGames().length }}</span>
          </h2>

          @if (allGames().length === 0 && pinnedGames().length === 0) {
            <div class="empty-state">
              <span class="empty-icon">🎮</span>
              <p>Noch keine Multiplayer-Spiele gefunden.</p>
              <p class="hint">Spiele werden geladen sobald Spieler sich anmelden.</p>
            </div>
          } @else {
            <div class="games-grid">
              @for (game of allGames(); track game.app_id) {
                <div class="game-card" (click)="openSteamStore(game.app_id)">
                  <div class="game-image">
                    <img [src]="game.header_image_url" [alt]="game.name" loading="lazy" />
                    <div class="owner-badge" [class.highlight]="game.owner_count >= 3">
                      {{ game.owner_count }}x
                    </div>
                  </div>
                  <div class="game-info">
                    <div class="game-title-row">
                      <h3>{{ game.name }}</h3>
                      <div class="price-review-column">
                        @if (game.price_formatted) {
                          <div class="price-badge" [class.free]="game.is_free" [class.discount]="game.discount_percent > 0">
                            @if (game.discount_percent > 0) {
                              <span class="discount-tag">-{{ game.discount_percent }}%</span>
                            }
                            @if (getPriceTier(game.price_cents, game.is_free)) {
                              <span class="price-tier">{{ getPriceTier(game.price_cents, game.is_free) }}</span>
                            }
                            <span class="price">{{ game.price_formatted }}</span>
                          </div>
                        }
                        @if (game.review_score >= 0) {
                          <div class="review-score" [class.positive]="game.review_score >= 85" [class.mixed]="game.review_score >= 70 && game.review_score < 85" [class.negative]="game.review_score < 70">
                            <span class="thumb">👍</span>
                            <span class="score">{{ game.review_score }}%</span>
                          </div>
                        }
                      </div>
                    </div>
                    <div class="game-meta">
                      <div class="owners has-tooltip">
                        <span class="owner-icon">👥</span>
                        <span>{{ game.owner_count }} {{ game.owner_count === 1 ? 'Besitzer' : 'Besitzer' }}</span>
                        <div class="owner-tooltip">
                          <div class="tooltip-title">Besitzer:</div>
                          @for (ownerName of getOwnerNameList(game.owners); track ownerName) {
                            <div class="tooltip-owner">{{ ownerName }}</div>
                          }
                        </div>
                      </div>
                      <div class="categories">
                        @for (cat of getMultiplayerCategories(game.categories); track cat) {
                          <span class="category-tag">{{ cat }}</span>
                        }
                      </div>
                    </div>
                  </div>
                </div>
              }
            </div>
          }
        </section>
      }
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .games-page {
      max-width: 1200px;
      margin: 0 auto;
      padding: 24px;
    }

    .sync-banner {
      background: linear-gradient(135deg, rgba($accent-primary, 0.15) 0%, rgba($accent-primary, 0.05) 100%);
      border: 1px solid rgba($accent-primary, 0.3);
      border-radius: 12px;
      padding: 16px 20px;
      margin-bottom: 24px;
      animation: fadeIn 0.3s ease;
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(-10px); }
      to { opacity: 1; transform: translateY(0); }
    }

    .sync-content {
      display: flex;
      align-items: center;
      gap: 16px;
    }

    .sync-spinner {
      width: 24px;
      height: 24px;
      border: 3px solid rgba($accent-primary, 0.3);
      border-top-color: $accent-primary;
      border-radius: 50%;
      animation: spin 1s linear infinite;
      flex-shrink: 0;
    }

    .sync-info {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }

    .sync-text {
      font-weight: 500;
      color: $text-primary;
    }

    .sync-progress {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .progress-bar {
      flex: 1;
      height: 6px;
      background: rgba($accent-primary, 0.2);
      border-radius: 3px;
      overflow: hidden;
      max-width: 200px;
    }

    .progress-fill {
      height: 100%;
      background: $accent-primary;
      border-radius: 3px;
      transition: width 0.3s ease;
    }

    .progress-text {
      font-size: 0.875rem;
      color: $text-secondary;
      font-variant-numeric: tabular-nums;
    }

    .current-game {
      font-size: 0.875rem;
      color: $text-secondary;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      max-width: 300px;
    }

    .page-header {
      text-align: center;
      margin-bottom: 32px;
      position: relative;

      h1 {
        font-size: 2rem;
        margin-bottom: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 12px;
      }

      .page-icon {
        font-size: 2.5rem;
      }

      .subtitle {
        color: $text-secondary;
        font-size: 1rem;
        margin-bottom: 16px;
      }

      .header-actions {
        display: flex;
        justify-content: center;
        margin-top: 16px;
      }

      .refresh-my-games-btn {
        background: linear-gradient(135deg, rgba($accent-primary, 0.15) 0%, rgba($accent-primary, 0.05) 100%);
        border: 1px solid rgba($accent-primary, 0.4);
        color: $text-primary;
        padding: 10px 20px;
        border-radius: 8px;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        font-size: 0.9375rem;
        font-weight: 500;
        transition: all 0.2s;

        &:hover:not(:disabled) {
          background: linear-gradient(135deg, rgba($accent-primary, 0.25) 0%, rgba($accent-primary, 0.1) 100%);
          border-color: $accent-primary;
          transform: translateY(-1px);
        }

        &:disabled {
          opacity: 0.6;
          cursor: not-allowed;
          transform: none;
        }

        .refresh-icon {
          font-size: 1rem;
        }

        .cooldown-icon {
          font-size: 1rem;
        }

        .btn-spinner {
          width: 16px;
          height: 16px;
          border: 2px solid rgba($accent-primary, 0.3);
          border-top-color: $accent-primary;
          border-radius: 50%;
          animation: spin 1s linear infinite;
        }
      }

      .header-buttons {
        display: flex;
        gap: 12px;
        justify-content: center;
        flex-wrap: wrap;
      }

      .refresh-btn {
        background: $bg-tertiary;
        border: 1px solid $border-color;
        color: $text-primary;
        padding: 8px 16px;
        border-radius: 8px;
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        gap: 8px;
        transition: all 0.2s;

        &:hover:not(:disabled) {
          background: $bg-secondary;
          border-color: $accent-primary;
        }

        &:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        &.admin-btn {
          border-color: $accent-warning;

          &:hover:not(:disabled) {
            background: rgba($accent-warning, 0.1);
            border-color: $accent-warning;
          }
        }

        .refresh-icon {
          display: inline-block;
          transition: transform 0.3s;

          &.spinning {
            animation: spin 1s linear infinite;
          }
        }
      }
    }

    @keyframes spin {
      from { transform: rotate(0deg); }
      to { transform: rotate(360deg); }
    }

    .loading {
      text-align: center;
      padding: 60px 20px;

      .spinner {
        width: 50px;
        height: 50px;
        border: 3px solid $border-color;
        border-top-color: $accent-primary;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin: 0 auto 16px;
      }

      p {
        color: $text-secondary;
      }
    }

    .error {
      text-align: center;
      padding: 60px 20px;
      background: rgba($accent-error, 0.1);
      border-radius: 12px;
      border: 1px solid rgba($accent-error, 0.3);

      .error-icon {
        font-size: 3rem;
        display: block;
        margin-bottom: 16px;
      }

      p {
        color: $text-primary;
        margin-bottom: 16px;
      }

      button {
        background: $accent-primary;
        color: white;
        border: none;
        padding: 10px 20px;
        border-radius: 8px;
        cursor: pointer;

        &:hover {
          opacity: 0.9;
        }
      }
    }

    .games-section {
      margin-bottom: 40px;

      h2 {
        display: flex;
        align-items: center;
        gap: 10px;
        font-size: 1.5rem;
        margin-bottom: 20px;
        padding-bottom: 12px;
        border-bottom: 2px solid $border-color;

        .section-icon {
          font-size: 1.5rem;
        }

        .count {
          background: $bg-tertiary;
          color: $text-secondary;
          font-size: 0.875rem;
          padding: 4px 10px;
          border-radius: 12px;
          font-weight: normal;
        }
      }

      &.pinned-section h2 {
        border-bottom-color: $accent-primary;
      }
    }

    .games-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 20px;
    }

    .game-card {
      background: $bg-secondary;
      border-radius: 12px;
      overflow: hidden;
      border: 1px solid $border-color;
      transition: all 0.2s;
      cursor: pointer;

      &:hover {
        box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
        border-color: $accent-primary;
      }

      &.pinned {
        border-color: $accent-primary;
        box-shadow: 0 0 20px rgba($accent-primary, 0.2);
      }

      .game-image {
        position: relative;
        aspect-ratio: 460 / 215;
        overflow: hidden;

        img {
          width: 100%;
          height: 100%;
          object-fit: cover;
          transition: transform 0.3s;
        }

        &:hover img {
          transform: scale(1.05);
        }

        .pinned-badge {
          position: absolute;
          top: 8px;
          left: 8px;
          background: rgba($accent-primary, 0.9);
          padding: 4px 8px;
          border-radius: 6px;
          font-size: 1rem;
        }

        .owner-badge {
          position: absolute;
          top: 8px;
          right: 8px;
          background: rgba(0, 0, 0, 0.8);
          color: $text-primary;
          padding: 4px 10px;
          border-radius: 6px;
          font-size: 0.875rem;
          font-weight: 600;

          &.highlight {
            background: $accent-success;
            color: white;
          }
        }

      }

      .game-info {
        padding: 16px;

        .game-title-row {
          display: flex;
          align-items: flex-start;
          justify-content: space-between;
          gap: 12px;
          margin-bottom: 10px;

          h3 {
            font-size: 1rem;
            line-height: 1.3;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
            margin: 0;
            flex: 1;
          }

          .price-badge {
            background: $bg-tertiary;
            color: $text-primary;
            padding: 4px 10px;
            border-radius: 6px;
            font-size: 0.875rem;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 6px;
            flex-shrink: 0;
            white-space: nowrap;

            &.free {
              background: rgba($accent-success, 0.9);
              color: white;
            }

            &.discount {
              .discount-tag {
                background: $accent-success;
                color: white;
                padding: 2px 6px;
                border-radius: 4px;
                font-size: 0.75rem;
                font-weight: 700;
              }
            }

            .price-tier {
              color: $accent-warning;
              font-size: 0.75rem;
              letter-spacing: -1px;
            }
          }

          .price-review-column {
            display: flex;
            flex-direction: column;
            align-items: flex-end;
            gap: 4px;
            flex-shrink: 0;
          }

          .review-score {
            display: flex;
            align-items: center;
            gap: 4px;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.875rem;
            font-weight: 600;

            .thumb {
              font-size: 0.7rem;
            }

            .score {
              font-weight: 700;
            }

            &.positive {
              background: rgba(76, 175, 80, 0.2);
              color: #4caf50;
            }

            &.mixed {
              background: rgba(255, 193, 7, 0.2);
              color: #ffc107;
            }

            &.negative {
              background: rgba(244, 67, 54, 0.2);
              color: #f44336;
            }
          }
        }

        .game-meta {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .owners {
          display: flex;
          align-items: center;
          gap: 6px;
          color: $text-secondary;
          font-size: 0.875rem;
          position: relative;

          .owner-icon {
            font-size: 1rem;
          }

          &.no-owners {
            color: $accent-error;
            opacity: 0.8;
          }

          &.has-tooltip {
            cursor: pointer;

            &:hover {
              color: $accent-primary;

              .owner-tooltip {
                opacity: 1;
                visibility: visible;
                transform: translateY(0);
              }
            }
          }

          .owner-tooltip {
            position: absolute;
            bottom: calc(100% + 8px);
            left: 0;
            background: $bg-tertiary;
            border: 1px solid $border-color;
            border-radius: 8px;
            padding: 12px;
            min-width: 150px;
            max-width: 250px;
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
            opacity: 0;
            visibility: hidden;
            transform: translateY(8px);
            transition: all 0.2s ease;
            z-index: 100;
            pointer-events: none;

            &::after {
              content: '';
              position: absolute;
              top: 100%;
              left: 16px;
              border: 8px solid transparent;
              border-top-color: $bg-tertiary;
            }

            &::before {
              content: '';
              position: absolute;
              top: 100%;
              left: 15px;
              border: 9px solid transparent;
              border-top-color: $border-color;
            }

            .tooltip-title {
              font-weight: 600;
              color: $text-primary;
              margin-bottom: 8px;
              font-size: 0.875rem;
            }

            .tooltip-owner {
              color: $text-secondary;
              font-size: 0.8125rem;
              padding: 2px 0;

              &:not(:last-child) {
                border-bottom: 1px solid rgba($border-color, 0.5);
                padding-bottom: 4px;
                margin-bottom: 4px;
              }
            }
          }
        }

        .categories {
          display: flex;
          flex-wrap: wrap;
          gap: 6px;
        }

        .category-tag {
          background: $bg-tertiary;
          color: $text-secondary;
          padding: 2px 8px;
          border-radius: 4px;
          font-size: 0.75rem;
        }
      }
    }

    .empty-state {
      text-align: center;
      padding: 60px 20px;
      background: $bg-secondary;
      border-radius: 12px;
      border: 1px dashed $border-color;

      .empty-icon {
        font-size: 4rem;
        display: block;
        margin-bottom: 16px;
        opacity: 0.5;
      }

      p {
        color: $text-secondary;
        margin-bottom: 8px;
      }

      .hint {
        font-size: 0.875rem;
        opacity: 0.7;
      }
    }

    // Responsive
    @media (max-width: 768px) {
      .games-page {
        padding: 16px;
      }

      .page-header h1 {
        font-size: 1.5rem;
      }

      .games-grid {
        grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
        gap: 16px;
      }

      .sync-banner {
        padding: 12px 16px;

        .sync-content {
          flex-direction: column;
          align-items: flex-start;
          gap: 12px;
        }

        .progress-bar {
          max-width: 100%;
        }
      }
    }
  `]
})
export class GamesComponent implements OnInit, OnDestroy {
  private gameService = inject(GameService);
  private userService = inject(UserService);
  private authService = inject(AuthService);
  private wsService = inject(WebSocketService);

  private subscriptions: Subscription[] = [];

  loading = signal(true);
  invalidating = signal(false);
  error = signal<string | null>(null);
  pinnedGames = signal<Game[]>([]);
  allGames = signal<Game[]>([]);
  users = signal<User[]>([]);

  // Sync status signals
  isSyncing = signal(false);
  syncPhase = signal<string>('');
  syncCurrentGame = signal<string>('');
  syncProcessed = signal(0);
  syncTotal = signal(0);
  syncPercentage = computed(() => {
    const total = this.syncTotal();
    if (total === 0) return 0;
    return Math.round((this.syncProcessed() / total) * 100);
  });

  // Refresh my games signals
  refreshingMyGames = signal(false);
  refreshCooldownRemaining = signal(0);
  private cooldownInterval: ReturnType<typeof setInterval> | null = null;

  isAdmin = computed(() => this.authService.user()?.is_admin ?? false);

  // Map of steamId -> username for displaying owner names
  private userMap = new Map<string, string>();

  ngOnInit() {
    this.loadUsers();
    this.loadGames();
    this.setupWebSocketListeners();
  }

  ngOnDestroy() {
    this.subscriptions.forEach(sub => sub.unsubscribe());
    if (this.cooldownInterval) {
      clearInterval(this.cooldownInterval);
    }
  }

  private setupWebSocketListeners() {
    // Listen for sync progress updates
    this.subscriptions.push(
      this.wsService.gamesSyncProgress$.subscribe(progress => {
        this.isSyncing.set(true);
        this.syncPhase.set(progress.phase);
        this.syncCurrentGame.set(progress.current_game);
        this.syncProcessed.set(progress.processed_count);
        this.syncTotal.set(progress.total_count);
      })
    );

    // Listen for sync complete
    this.subscriptions.push(
      this.wsService.gamesSyncComplete$.subscribe(() => {
        this.isSyncing.set(false);
        this.syncPhase.set('');
        this.syncCurrentGame.set('');
        this.syncProcessed.set(0);
        this.syncTotal.set(0);
        // Reload games to get updated data
        this.loadGamesQuietly();
      })
    );
  }

  loadUsers() {
    this.userService.getAll().subscribe({
      next: (users) => {
        this.users.set(users);
        users.forEach(u => this.userMap.set(u.steam_id, u.username));
      },
      error: (err) => console.error('Failed to load users', err)
    });
  }

  invalidateCache() {
    this.invalidating.set(true);
    this.error.set(null);

    this.gameService.invalidateCache().subscribe({
      next: () => {
        this.invalidating.set(false);
        // Now refresh to get fresh data from Steam
        this.refreshGames();
      },
      error: (err) => {
        console.error('Failed to invalidate cache', err);
        this.error.set('Fehler beim Invalidieren des Caches.');
        this.invalidating.set(false);
      }
    });
  }

  loadGames() {
    this.loading.set(true);
    this.error.set(null);

    this.gameService.getMultiplayerGames().subscribe({
      next: (response) => {
        this.pinnedGames.set(response.pinned_games || []);
        this.allGames.set(response.all_games || []);
        this.loading.set(false);

        // Check sync status and trigger background sync if needed
        if (response.sync_status) {
          if (response.sync_status.is_syncing) {
            this.isSyncing.set(true);
            this.syncPhase.set(response.sync_status.phase);
            this.syncCurrentGame.set(response.sync_status.current_game);
            this.syncProcessed.set(response.sync_status.processed);
            this.syncTotal.set(response.sync_status.total);
          } else if (response.sync_status.needs_sync) {
            // Trigger background sync
            this.startBackgroundSync();
          }
        }
      },
      error: (err) => {
        console.error('Failed to load games', err);
        this.error.set('Fehler beim Laden der Spiele. Bitte versuche es erneut.');
        this.loading.set(false);
      }
    });
  }

  // Load games without showing loading spinner (used after sync complete)
  private loadGamesQuietly() {
    this.gameService.getMultiplayerGames().subscribe({
      next: (response) => {
        this.pinnedGames.set(response.pinned_games || []);
        this.allGames.set(response.all_games || []);
      },
      error: (err) => console.error('Failed to reload games after sync', err)
    });
  }

  private startBackgroundSync() {
    this.gameService.startBackgroundSync().subscribe({
      next: () => {
        console.log('Background sync started');
      },
      error: (err) => {
        // 409 Conflict means sync is already in progress
        if (err.status !== 409) {
          console.error('Failed to start background sync', err);
        }
      }
    });
  }

  refreshGames() {
    this.loading.set(true);
    this.error.set(null);

    this.gameService.refreshGames().subscribe({
      next: (response) => {
        this.pinnedGames.set(response.pinned_games || []);
        this.allGames.set(response.all_games || []);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Failed to refresh games', err);
        this.error.set('Fehler beim Aktualisieren der Spiele.');
        this.loading.set(false);
      }
    });
  }

  getOwnerNames(owners: string[]): string {
    if (!owners || owners.length === 0) return 'Keine Besitzer';
    return owners
      .map(steamId => this.userMap.get(steamId) || steamId)
      .join(', ');
  }

  getOwnerNameList(owners: string[]): string[] {
    if (!owners || owners.length === 0) return [];
    return owners.map(steamId => this.userMap.get(steamId) || steamId);
  }

  getMultiplayerCategories(categories: string[]): string[] {
    const mpCategories = ['Multi-player', 'Co-op', 'Online Co-op', 'LAN Co-op', 'LAN PvP', 'Online PvP', 'PvP'];
    return (categories || []).filter(cat => mpCategories.includes(cat));
  }

  getPriceTier(priceCents: number, isFree: boolean): string {
    if (isFree || priceCents === 0) return '';
    const priceEuros = priceCents / 100;
    if (priceEuros < 10) return '€';
    if (priceEuros < 20) return '€€';
    return '€€€';
  }

  openSteamStore(appId: number) {
    window.open(`https://store.steampowered.com/app/${appId}`, '_blank');
  }

  refreshMyGames() {
    if (this.refreshingMyGames() || this.refreshCooldownRemaining() > 0) {
      return;
    }

    this.refreshingMyGames.set(true);
    this.gameService.refreshMyGames().subscribe({
      next: (response) => {
        this.refreshingMyGames.set(false);
        // Reload games list to show updated data
        this.loadGamesQuietly();
        // Start cooldown (5 minutes)
        this.startCooldown(5 * 60);
      },
      error: (err) => {
        this.refreshingMyGames.set(false);
        if (err.status === 429 && err.error?.remaining_seconds) {
          // Server says we're on cooldown
          this.startCooldown(err.error.remaining_seconds);
        } else {
          console.error('Failed to refresh my games', err);
        }
      }
    });
  }

  private startCooldown(seconds: number) {
    this.refreshCooldownRemaining.set(seconds);

    if (this.cooldownInterval) {
      clearInterval(this.cooldownInterval);
    }

    this.cooldownInterval = setInterval(() => {
      const remaining = this.refreshCooldownRemaining();
      if (remaining <= 1) {
        this.refreshCooldownRemaining.set(0);
        if (this.cooldownInterval) {
          clearInterval(this.cooldownInterval);
          this.cooldownInterval = null;
        }
      } else {
        this.refreshCooldownRemaining.set(remaining - 1);
      }
    }, 1000);
  }

  formatCooldown(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  getRefreshButtonTitle(): string {
    if (this.refreshingMyGames()) {
      return 'Aktualisiere deine Spielebibliothek...';
    }
    if (this.refreshCooldownRemaining() > 0) {
      return `Bitte warte noch ${this.formatCooldown(this.refreshCooldownRemaining())}`;
    }
    return 'Aktualisiere deine Steam-Spielebibliothek';
  }
}
