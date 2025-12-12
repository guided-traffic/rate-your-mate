import { Component, OnInit, signal, inject, ViewChild, ElementRef, AfterViewChecked } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { SettingsService, AdminUserInfo, BannedUser } from '../../services/settings.service';
import { AuthService } from '../../services/auth.service';
import { NotificationService } from '../../services/notification.service';
import { GameService } from '../../services/game.service';

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="admin-page">
      <div class="admin-container">
        <!-- Password Gate -->
        @if (!authenticated()) {
          <div class="password-gate">
            <div class="password-card">
              <div class="password-header">
                <span class="lock-icon">🔒</span>
                <h2>Admin-Bereich</h2>
                <p>Bitte gib das Admin-Passwort ein, um fortzufahren.</p>
              </div>

              @if (checkingPassword()) {
                <div class="loading-inline">
                  <div class="spinner"></div>
                  <span>Prüfe...</span>
                </div>
              } @else {
                <form (ngSubmit)="submitPassword()" class="password-form">
                  <div class="input-wrapper">
                    <input
                      #passwordField
                      type="password"
                      [(ngModel)]="passwordInput"
                      name="password"
                      placeholder="Admin-Passwort"
                      class="password-input"
                      [class.error]="passwordError()"
                    />
                    @if (passwordError()) {
                      <span class="error-text">{{ passwordError() }}</span>
                    }
                  </div>
                  <div class="password-actions">
                    <button type="button" (click)="goBack()" class="back-btn">
                      ← Zurück
                    </button>
                    <button type="submit" [disabled]="!passwordInput" class="submit-btn">
                      🔓 Entsperren
                    </button>
                  </div>
                </form>
              }
            </div>
          </div>
        } @else {
          <!-- Admin Panel Content -->
          <div class="admin-header">
            <h1>⚙️ Admin Panel</h1>
            <p class="admin-subtitle">Einstellungen für das Credit System</p>
          </div>

          @if (loading()) {
            <div class="loading">
              <div class="spinner"></div>
              <span>Lade Einstellungen...</span>
            </div>
          } @else if (error()) {
            <div class="error-message">
              <span>❌</span>
              <p>{{ error() }}</p>
              <button (click)="loadSettings()" class="retry-btn">Erneut versuchen</button>
            </div>
          } @else {
            <div class="settings-card">
              <div class="setting-group">
                <label for="creditInterval">Credit Interval (Minuten)</label>
                <p class="setting-description">
                  Wie viele Minuten zwischen dem Verdienen von Credits vergehen.
                </p>
                <div class="input-group">
                  <input
                    type="number"
                    id="creditInterval"
                    [(ngModel)]="creditIntervalMinutes"
                    min="1"
                    max="60"
                    class="setting-input"
                  />
                  <span class="input-suffix">min</span>
                </div>
              </div>

              <div class="setting-group">
                <label for="creditMax">Maximale Credits</label>
                <p class="setting-description">
                  Die maximale Anzahl an Credits, die ein Spieler ansammeln kann.
                </p>
                <div class="input-group">
                  <input
                    type="number"
                    id="creditMax"
                    [(ngModel)]="creditMax"
                    min="1"
                    max="100"
                    class="setting-input"
                  />
                  <span class="input-suffix">Credits</span>
                </div>
              </div>

              <div class="actions">
                <button
                  (click)="saveSettings()"
                  [disabled]="saving() || !hasChanges()"
                  class="save-btn"
                >
                  @if (saving()) {
                    <span class="btn-spinner"></span>
                    Speichern...
                  } @else {
                    💾 Einstellungen speichern
                  }
                </button>
                <button
                  (click)="resetToOriginal()"
                  [disabled]="saving() || !hasChanges()"
                  class="reset-btn"
                >
                  ↩️ Zurücksetzen
                </button>
              </div>

              @if (hasChanges()) {
                <div class="changes-notice">
                  <span>⚠️</span>
                  <span>Du hast ungespeicherte Änderungen.</span>
                </div>
              }
            </div>

            <div class="info-card">
              <h3>ℹ️ Hinweis</h3>
              <p>
                Änderungen werden <strong>sofort live</strong> an alle verbundenen Spieler übertragen.
                Die Credits-Anzeige aller Spieler wird automatisch aktualisiert.
              </p>
            </div>

            <div class="voting-control-card" [class.paused]="votingPaused()">
              <div class="voting-status">
                <span class="status-icon">{{ votingPaused() ? '⏸️' : '▶️' }}</span>
                <div class="status-text">
                  <h3>Voting Status</h3>
                  <p>{{ votingPaused() ? 'Voting ist pausiert - niemand kann bewerten' : 'Voting ist aktiv' }}</p>
                </div>
              </div>
              <button
                (click)="toggleVotingPause()"
                [disabled]="togglingPause()"
                class="toggle-pause-btn"
                [class.paused]="votingPaused()"
              >
                @if (togglingPause()) {
                  <span class="btn-spinner"></span>
                } @else if (votingPaused()) {
                  ▶️ Voting fortsetzen
                } @else {
                  ⏸️ Voting pausieren
                }
              </button>
            </div>

            <div class="visibility-card">
              <h3>👁️ Abstimmungs-Sichtbarkeit</h3>
              <p class="action-description">
                Steuere, ob Abstimmungen anonym oder öffentlich angezeigt werden.
              </p>
              <div class="visibility-options">
                <label class="visibility-option" [class.active]="voteVisibilityMode() === 'all_secret'">
                  <input
                    type="radio"
                    name="visibility"
                    value="all_secret"
                    [checked]="voteVisibilityMode() === 'all_secret'"
                    (change)="setVoteVisibilityMode('all_secret')"
                    [disabled]="updatingVisibility()"
                  />
                  <span class="option-icon">🕵️</span>
                  <span class="option-text">
                    <strong>Alles geheim</strong>
                    <small>Alle Abstimmungen sind anonym</small>
                  </span>
                </label>
                <label class="visibility-option" [class.active]="voteVisibilityMode() === 'user_choice'">
                  <input
                    type="radio"
                    name="visibility"
                    value="user_choice"
                    [checked]="voteVisibilityMode() === 'user_choice'"
                    (change)="setVoteVisibilityMode('user_choice')"
                    [disabled]="updatingVisibility()"
                  />
                  <span class="option-icon">🎯</span>
                  <span class="option-text">
                    <strong>Nutzer-Wahl</strong>
                    <small>Spieler entscheiden selbst</small>
                  </span>
                </label>
                <label class="visibility-option" [class.active]="voteVisibilityMode() === 'all_public'">
                  <input
                    type="radio"
                    name="visibility"
                    value="all_public"
                    [checked]="voteVisibilityMode() === 'all_public'"
                    (change)="setVoteVisibilityMode('all_public')"
                    [disabled]="updatingVisibility()"
                  />
                  <span class="option-icon">👁️</span>
                  <span class="option-text">
                    <strong>Alles offen</strong>
                    <small>Alle Abstimmungen sind sichtbar</small>
                  </span>
                </label>
              </div>
              @if (updatingVisibility()) {
                <div class="loading-inline visibility-loading">
                  <div class="spinner"></div>
                  <span>Wird aktualisiert...</span>
                </div>
              }
            </div>

            <div class="credit-actions-card">
              <h3>💰 Credit Aktionen</h3>
              <p class="action-description">
                Manuelle Credit-Verwaltung für alle Spieler gleichzeitig.
              </p>
              <div class="credit-actions">
                <button
                  (click)="giveEveryoneCredit()"
                  [disabled]="givingCredits()"
                  class="give-credit-btn"
                >
                  @if (givingCredits()) {
                    <span class="btn-spinner"></span>
                    Wird verteilt...
                  } @else {
                    🎁 Jedem 1 Credit geben
                  }
                </button>
                <button
                  (click)="resetAllCredits()"
                  [disabled]="resettingCredits()"
                  class="reset-credits-btn"
                >
                  @if (resettingCredits()) {
                    <span class="btn-spinner"></span>
                    Wird zurückgesetzt...
                  } @else {
                    🔄 Alle Credits auf 0 setzen
                  }
                </button>
              </div>
            </div>

            <div class="steam-actions-card">
              <h3>☁️ Steam Aktionen</h3>
              <p class="action-description">
                Spiele-Daten von Steam neu laden (Cache invalidieren).
              </p>
              <div class="steam-actions">
                <button
                  (click)="invalidateSteamCache()"
                  [disabled]="invalidatingCache()"
                  class="steam-update-btn"
                >
                  @if (invalidatingCache()) {
                    <span class="btn-spinner"></span>
                    Wird aktualisiert...
                  } @else {
                    ☁️ Update von Steam
                  }
                </button>
              </div>
            </div>

            <!-- Player Management Section -->
            <div class="player-management-card">
              <h3>👥 Spielerverwaltung</h3>
              <p class="action-description">
                Alle angemeldeten Spieler verwalten. Kicken löscht alle Daten, Bannen verhindert zusätzlich die Wiederkehr.
              </p>

              @if (loadingUsers()) {
                <div class="loading-inline">
                  <div class="spinner"></div>
                  <span>Lade Spieler...</span>
                </div>
              } @else if (userList().length === 0) {
                <p class="no-users">Keine Spieler angemeldet.</p>
              } @else {
                <div class="user-list">
                  @for (user of userList(); track user.id) {
                    <div class="user-item" [class.confirming]="confirmingAction()?.userId === user.id">
                      <div class="user-info">
                        <img [src]="user.avatar_small || 'assets/default-avatar.png'" [alt]="user.username" class="user-avatar" />
                        <div class="user-details">
                          <span class="user-name">{{ user.username }}</span>
                          <span class="user-steam-id">{{ user.steam_id }}</span>
                        </div>
                      </div>
                      <div class="user-actions">
                        @if (confirmingAction()?.userId === user.id) {
                          <div class="confirm-inline">
                            <span class="confirm-text">
                              {{ confirmingAction()?.action === 'kick' ? 'Kicken?' : 'Bannen?' }}
                            </span>
                            <button (click)="cancelUserAction()" class="cancel-sm-btn">✖️</button>
                            <button
                              (click)="executeUserAction()"
                              [disabled]="executingAction()"
                              class="confirm-sm-btn"
                              [class.ban]="confirmingAction()?.action === 'ban'"
                            >
                              @if (executingAction()) {
                                <span class="btn-spinner-sm"></span>
                              } @else {
                                ✓
                              }
                            </button>
                          </div>
                        } @else {
                          <button
                            (click)="startKickUser(user)"
                            [disabled]="executingAction()"
                            class="kick-btn"
                            title="Kicken (kann sich wieder anmelden)"
                          >
                            👢
                          </button>
                          @if (!isCurrentUser(user)) {
                            <button
                              (click)="startBanUser(user)"
                              [disabled]="executingAction()"
                              class="ban-btn"
                              title="Bannen (kann sich nicht wieder anmelden)"
                            >
                              🚫
                            </button>
                          }
                        }
                      </div>
                    </div>
                  }
                </div>
              }

              <!-- Banned Users Section -->
              @if (bannedUsers().length > 0) {
                <div class="banned-section">
                  <h4>🚫 Gebannte Spieler</h4>
                  <div class="banned-list">
                    @for (banned of bannedUsers(); track banned.id) {
                      <div class="banned-item">
                        <div class="banned-info">
                          <span class="banned-name">{{ banned.username }}</span>
                          <span class="banned-steam-id">{{ banned.steam_id }}</span>
                          @if (banned.reason) {
                            <span class="banned-reason">Grund: {{ banned.reason }}</span>
                          }
                        </div>
                        <button
                          (click)="unbanUser(banned)"
                          [disabled]="executingAction()"
                          class="unban-btn"
                          title="Entbannen"
                        >
                          ✅ Entbannen
                        </button>
                      </div>
                    }
                  </div>
                </div>
              }
            </div>

            <div class="danger-zone-card">
              <h3>⚠️ Gefahrenzone</h3>
              <p class="action-description">
                Vorsicht! Diese Aktionen können nicht rückgängig gemacht werden.
              </p>
              <div class="danger-actions">
                @if (!confirmingDeleteVotes()) {
                  <button
                    (click)="startDeleteVotesConfirmation()"
                    [disabled]="deletingVotes()"
                    class="danger-btn"
                  >
                    🗑️ Alle Votes löschen
                  </button>
                } @else {
                  <div class="confirm-delete-container">
                    <p class="confirm-warning">
                      ⚠️ Bist du sicher? Alle Votes und das Leaderboard werden gelöscht!
                    </p>
                    <div class="confirm-actions">
                      <button
                        (click)="cancelDeleteVotes()"
                        class="cancel-btn"
                      >
                        ✖️ Abbrechen
                      </button>
                      <button
                        (click)="confirmDeleteAllVotes()"
                        [disabled]="deletingVotes()"
                        class="confirm-danger-btn"
                      >
                        @if (deletingVotes()) {
                          <span class="btn-spinner"></span>
                          Wird gelöscht...
                        } @else {
                          ✓ Ja, alle Votes löschen
                        }
                      </button>
                    </div>
                  </div>
                }
              </div>
            </div>
          }
        }
      </div>
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .admin-page {
      min-height: calc(100vh - 64px);
      padding: 32px 24px;
      background: $bg-primary;
    }

    .admin-container {
      max-width: 600px;
      margin: 0 auto;
    }

    /* Password Gate Styles */
    .password-gate {
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: calc(100vh - 200px);
    }

    .password-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      padding: 40px;
      width: 100%;
      max-width: 400px;
      text-align: center;
    }

    .password-header {
      margin-bottom: 32px;

      .lock-icon {
        font-size: 48px;
        display: block;
        margin-bottom: 16px;
      }

      h2 {
        font-size: 24px;
        font-weight: 700;
        color: $text-primary;
        margin-bottom: 8px;
      }

      p {
        font-size: 14px;
        color: $text-secondary;
      }
    }

    .loading-inline {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 12px;
      padding: 24px;
      color: $text-secondary;
    }

    .password-form {
      display: flex;
      flex-direction: column;
      gap: 20px;
    }

    .input-wrapper {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .password-input {
      width: 100%;
      padding: 14px 18px;
      background: $bg-tertiary;
      border: 2px solid $border-color;
      border-radius: $radius-md;
      color: $text-primary;
      font-size: 16px;
      text-align: center;
      letter-spacing: 2px;
      transition: all $transition-fast;

      &::placeholder {
        letter-spacing: normal;
        color: $text-muted;
      }

      &:focus {
        outline: none;
        border-color: $accent-primary;
        box-shadow: 0 0 0 3px rgba($accent-primary, 0.2);
      }

      &.error {
        border-color: $accent-error;
        box-shadow: 0 0 0 3px rgba($accent-error, 0.2);
      }
    }

    .error-text {
      color: $accent-error;
      font-size: 13px;
    }

    .password-actions {
      display: flex;
      gap: 12px;
    }

    .back-btn {
      flex: 1;
      padding: 14px 20px;
      background: $bg-tertiary;
      color: $text-secondary;
      border: 1px solid $border-color;
      border-radius: $radius-md;
      font-size: 15px;
      font-weight: 500;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover {
        background: $bg-hover;
        color: $text-primary;
      }
    }

    .submit-btn {
      flex: 2;
      padding: 14px 24px;
      background: $gradient-primary;
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: $shadow-lg;
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    /* Admin Panel Styles */
    .admin-header {
      text-align: center;
      margin-bottom: 32px;

      h1 {
        font-size: 28px;
        font-weight: 700;
        color: $text-primary;
        margin-bottom: 8px;
      }

      .admin-subtitle {
        color: $text-secondary;
        font-size: 16px;
      }
    }

    .loading {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 16px;
      padding: 48px;
      color: $text-secondary;
    }

    .spinner {
      width: 40px;
      height: 40px;
      border: 3px solid $border-color;
      border-top-color: $accent-primary;
      border-radius: 50%;
      animation: spin 1s linear infinite;
    }

    .error-message {
      text-align: center;
      padding: 48px;
      background: $bg-card;
      border-radius: $radius-lg;
      border: 1px solid $accent-error;

      span {
        font-size: 48px;
        display: block;
        margin-bottom: 16px;
      }

      p {
        color: $accent-error;
        margin-bottom: 16px;
      }

      .retry-btn {
        padding: 10px 20px;
        background: $accent-primary;
        color: white;
        border: none;
        border-radius: $radius-md;
        cursor: pointer;
        font-weight: 500;

        &:hover {
          background: $accent-secondary;
        }
      }
    }

    .settings-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      padding: 24px;
      margin-bottom: 24px;
    }

    .setting-group {
      margin-bottom: 24px;

      &:last-of-type {
        margin-bottom: 32px;
      }

      label {
        display: block;
        font-size: 16px;
        font-weight: 600;
        color: $text-primary;
        margin-bottom: 4px;
      }

      .setting-description {
        font-size: 14px;
        color: $text-muted;
        margin-bottom: 12px;
      }
    }

    .input-group {
      display: flex;
      align-items: center;
      gap: 8px;

      .setting-input {
        width: 120px;
        padding: 12px 16px;
        background: $bg-tertiary;
        border: 1px solid $border-color;
        border-radius: $radius-md;
        color: $text-primary;
        font-size: 18px;
        font-weight: 600;

        &:focus {
          outline: none;
          border-color: $accent-primary;
          box-shadow: 0 0 0 3px rgba($accent-primary, 0.2);
        }

        &::-webkit-inner-spin-button,
        &::-webkit-outer-spin-button {
          opacity: 1;
        }
      }

      .input-suffix {
        color: $text-secondary;
        font-size: 14px;
      }
    }

    .actions {
      display: flex;
      gap: 12px;
    }

    .save-btn {
      flex: 1;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 14px 24px;
      background: $gradient-primary;
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 16px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: $shadow-lg;
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .reset-btn {
      padding: 14px 20px;
      background: $bg-tertiary;
      color: $text-secondary;
      border: 1px solid $border-color;
      border-radius: $radius-md;
      font-size: 14px;
      font-weight: 500;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        background: $bg-hover;
        color: $text-primary;
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .btn-spinner {
      width: 16px;
      height: 16px;
      border: 2px solid rgba(white, 0.3);
      border-top-color: white;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    .changes-notice {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-top: 16px;
      padding: 12px;
      background: rgba($accent-warning, 0.1);
      border: 1px solid rgba($accent-warning, 0.3);
      border-radius: $radius-md;
      color: $accent-warning;
      font-size: 14px;
    }

    .info-card {
      background: rgba($accent-primary, 0.1);
      border: 1px solid rgba($accent-primary, 0.2);
      border-radius: $radius-lg;
      padding: 20px;
      margin-bottom: 24px;

      h3 {
        font-size: 16px;
        font-weight: 600;
        color: $accent-primary;
        margin-bottom: 8px;
      }

      p {
        font-size: 14px;
        color: $text-secondary;
        line-height: 1.6;

        strong {
          color: $accent-primary;
        }
      }
    }

    .credit-actions-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      padding: 24px;
      margin-bottom: 24px;

      h3 {
        font-size: 18px;
        font-weight: 600;
        color: $text-primary;
        margin-bottom: 8px;
      }

      .action-description {
        font-size: 14px;
        color: $text-muted;
        margin-bottom: 20px;
      }
    }

    .steam-actions-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      padding: 24px;

      h3 {
        font-size: 18px;
        font-weight: 600;
        color: $text-primary;
        margin-bottom: 8px;
      }

      .action-description {
        font-size: 14px;
        color: $text-muted;
        margin-bottom: 20px;
      }
    }

    .steam-actions {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
    }

    .steam-update-btn {
      flex: 1;
      min-width: 200px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 14px 24px;
      background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .danger-zone-card {
      background: $bg-card;
      border: 2px solid $accent-error;
      border-radius: $radius-lg;
      padding: 24px;
      margin-top: 24px;

      h3 {
        font-size: 18px;
        font-weight: 600;
        color: $accent-error;
        margin-bottom: 8px;
      }

      .action-description {
        font-size: 14px;
        color: $text-muted;
        margin-bottom: 20px;
      }
    }

    .danger-actions {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .danger-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 14px 24px;
      background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .confirm-delete-container {
      background: rgba($accent-error, 0.1);
      border: 1px solid rgba($accent-error, 0.3);
      border-radius: $radius-md;
      padding: 20px;
    }

    .confirm-warning {
      color: $accent-error;
      font-weight: 600;
      margin-bottom: 16px;
      text-align: center;
    }

    .confirm-actions {
      display: flex;
      gap: 12px;
      justify-content: center;
    }

    .cancel-btn {
      padding: 12px 24px;
      background: $bg-tertiary;
      color: $text-secondary;
      border: 1px solid $border-color;
      border-radius: $radius-md;
      font-size: 14px;
      font-weight: 500;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover {
        background: $bg-hover;
        color: $text-primary;
      }
    }

    .confirm-danger-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 12px 24px;
      background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .credit-actions {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
    }

    .give-credit-btn {
      flex: 1;
      min-width: 200px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 14px 24px;
      background: linear-gradient(135deg, #10b981 0%, #059669 100%);
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .reset-credits-btn {
      flex: 1;
      min-width: 200px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 14px 24px;
      background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .voting-control-card {
      background: $bg-card;
      border: 2px solid #10b981;
      border-radius: $radius-lg;
      padding: 24px;
      margin-bottom: 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;

      &.paused {
        border-color: $accent-warning;
        background: rgba($accent-warning, 0.05);
      }
    }

    .voting-status {
      display: flex;
      align-items: center;
      gap: 16px;

      .status-icon {
        font-size: 32px;
      }

      .status-text {
        h3 {
          font-size: 16px;
          font-weight: 600;
          color: $text-primary;
          margin-bottom: 4px;
        }

        p {
          font-size: 14px;
          color: $text-secondary;
        }
      }
    }

    .toggle-pause-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 12px 24px;
      background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
      color: white;
      border: none;
      border-radius: $radius-md;
      font-size: 15px;
      font-weight: 600;
      cursor: pointer;
      transition: all $transition-fast;
      white-space: nowrap;

      &:hover:not(:disabled) {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px rgba(245, 158, 11, 0.4);
      }

      &.paused {
        background: linear-gradient(135deg, #10b981 0%, #059669 100%);

        &:hover:not(:disabled) {
          box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
        }
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    /* Visibility Mode Card */
    .visibility-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      padding: 24px;
      margin-bottom: 24px;

      h3 {
        font-size: 18px;
        font-weight: 600;
        color: $text-primary;
        margin-bottom: 8px;
      }
    }

    .visibility-options {
      display: flex;
      flex-direction: column;
      gap: 12px;
      margin-top: 16px;
    }

    .visibility-option {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 16px;
      background: $bg-tertiary;
      border: 2px solid $border-color;
      border-radius: $radius-md;
      cursor: pointer;
      transition: all $transition-fast;

      input[type="radio"] {
        display: none;
      }

      .option-icon {
        font-size: 24px;
      }

      .option-text {
        display: flex;
        flex-direction: column;
        gap: 2px;

        strong {
          font-size: 15px;
          color: $text-primary;
        }

        small {
          font-size: 13px;
          color: $text-secondary;
        }
      }

      &:hover:not(.active) {
        border-color: $border-light;
        background: rgba($accent-primary, 0.05);
      }

      &.active {
        border-color: $accent-primary;
        background: rgba($accent-primary, 0.1);

        .option-text strong {
          color: $accent-primary;
        }
      }
    }

    .visibility-loading {
      margin-top: 12px;
      justify-content: center;
    }

    /* Player Management Styles */
    .player-management-card {
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      padding: 24px;
      margin-bottom: 24px;

      h3 {
        font-size: 18px;
        font-weight: 600;
        color: $text-primary;
        margin-bottom: 8px;
      }

      h4 {
        font-size: 16px;
        font-weight: 600;
        color: $text-primary;
        margin: 24px 0 12px 0;
      }
    }

    .user-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      margin-top: 16px;
    }

    .user-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      background: $bg-tertiary;
      border-radius: $radius-md;
      transition: all $transition-fast;

      &.confirming {
        background: rgba($accent-warning, 0.1);
        border: 1px solid rgba($accent-warning, 0.3);
      }
    }

    .user-info {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .user-avatar {
      width: 36px;
      height: 36px;
      border-radius: 50%;
      object-fit: cover;
    }

    .user-details {
      display: flex;
      flex-direction: column;
    }

    .user-name {
      font-weight: 600;
      color: $text-primary;
      font-size: 14px;
    }

    .user-steam-id {
      font-size: 12px;
      color: $text-muted;
      font-family: monospace;
    }

    .user-actions {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .kick-btn, .ban-btn {
      width: 36px;
      height: 36px;
      display: flex;
      align-items: center;
      justify-content: center;
      border: none;
      border-radius: $radius-md;
      font-size: 16px;
      cursor: pointer;
      transition: all $transition-fast;

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .kick-btn {
      background: rgba($accent-warning, 0.2);
      color: $accent-warning;

      &:hover:not(:disabled) {
        background: rgba($accent-warning, 0.3);
      }
    }

    .ban-btn {
      background: rgba($accent-error, 0.2);
      color: $accent-error;

      &:hover:not(:disabled) {
        background: rgba($accent-error, 0.3);
      }
    }

    .confirm-inline {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .confirm-text {
      font-size: 13px;
      font-weight: 600;
      color: $accent-warning;
    }

    .cancel-sm-btn, .confirm-sm-btn {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      border: none;
      border-radius: 6px;
      font-size: 12px;
      cursor: pointer;
      transition: all $transition-fast;
    }

    .cancel-sm-btn {
      background: $bg-tertiary;
      color: $text-secondary;

      &:hover {
        background: $bg-hover;
      }
    }

    .confirm-sm-btn {
      background: $accent-warning;
      color: white;

      &.ban {
        background: $accent-error;
      }

      &:hover:not(:disabled) {
        opacity: 0.9;
      }

      &:disabled {
        opacity: 0.5;
      }
    }

    .btn-spinner-sm {
      width: 12px;
      height: 12px;
      border: 2px solid rgba(white, 0.3);
      border-top-color: white;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    .banned-section {
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid $border-color;
    }

    .banned-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .banned-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      background: rgba($accent-error, 0.1);
      border: 1px solid rgba($accent-error, 0.2);
      border-radius: $radius-md;
    }

    .banned-info {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .banned-name {
      font-weight: 600;
      color: $text-primary;
      font-size: 14px;
    }

    .banned-steam-id {
      font-size: 12px;
      color: $text-muted;
      font-family: monospace;
    }

    .banned-reason {
      font-size: 12px;
      color: $text-secondary;
      font-style: italic;
    }

    .unban-btn {
      padding: 8px 16px;
      background: rgba($accent-success, 0.2);
      color: $accent-success;
      border: none;
      border-radius: $radius-md;
      font-size: 13px;
      font-weight: 500;
      cursor: pointer;
      transition: all $transition-fast;

      &:hover:not(:disabled) {
        background: rgba($accent-success, 0.3);
      }

      &:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }
    }

    .no-users {
      text-align: center;
      color: $text-secondary;
      padding: 24px;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }
  `]
})
export class AdminComponent implements OnInit, AfterViewChecked {
  private settingsService = inject(SettingsService);
  private authService = inject(AuthService);
  private notifications = inject(NotificationService);
  private router = inject(Router);
  private gameService = inject(GameService);

  @ViewChild('passwordField') passwordField?: ElementRef<HTMLInputElement>;
  private shouldFocusPassword = false;

  // Password gate
  authenticated = signal(false);
  checkingPassword = signal(false);
  passwordError = signal<string | null>(null);
  passwordInput = '';

  loading = signal(true);
  saving = signal(false);
  error = signal<string | null>(null);
  resettingCredits = signal(false);
  givingCredits = signal(false);
  togglingPause = signal(false);
  votingPaused = signal(false);
  invalidatingCache = signal(false);
  deletingVotes = signal(false);
  confirmingDeleteVotes = signal(false);
  voteVisibilityMode = signal<'user_choice' | 'all_secret' | 'all_public'>('user_choice');
  updatingVisibility = signal(false);

  // Player management
  allUsers = signal<AdminUserInfo[]>([]);
  bannedUsers = signal<BannedUser[]>([]);
  loadingUsers = signal(false);
  loadingBannedUsers = signal(false);
  confirmingAction = signal<{ userId: number; username: string; action: 'kick' | 'ban' } | null>(null);
  executingAction = signal(false);

  // Computed signal for template
  userList = this.allUsers;

  // Form values
  creditIntervalMinutes = 10;
  creditMax = 10;

  // Original values for comparison
  private originalCreditIntervalMinutes = 10;
  private originalCreditMax = 10;

  ngAfterViewChecked(): void {
    if (this.shouldFocusPassword && this.passwordField) {
      this.passwordField.nativeElement.focus();
      this.shouldFocusPassword = false;
    }
  }

  ngOnInit(): void {
    // Check if user is admin
    const user = this.authService.user();
    if (!user?.is_admin) {
      this.router.navigate(['/timeline']);
      return;
    }

    // Check if password is required
    this.checkPasswordRequired();
  }

  checkPasswordRequired(): void {
    this.checkingPassword.set(true);
    this.settingsService.checkAdminPasswordRequired().subscribe({
      next: (response) => {
        this.checkingPassword.set(false);
        if (!response.password_required) {
          // No password required, directly authenticate
          this.authenticated.set(true);
          this.loadSettings();
        } else {
          // Password required, focus the input field
          this.shouldFocusPassword = true;
        }
      },
      error: (err) => {
        console.error('Failed to check password requirement:', err);
        this.checkingPassword.set(false);
        // On error, assume password is required for security
        this.shouldFocusPassword = true;
      }
    });
  }

  submitPassword(): void {
    if (!this.passwordInput) return;

    this.checkingPassword.set(true);
    this.passwordError.set(null);

    this.settingsService.verifyAdminPassword(this.passwordInput).subscribe({
      next: (response) => {
        this.checkingPassword.set(false);
        if (response.valid) {
          this.authenticated.set(true);
          this.passwordInput = '';
          this.loadSettings();
        } else {
          this.passwordError.set('Falsches Passwort');
        }
      },
      error: (err) => {
        this.checkingPassword.set(false);
        if (err.status === 403) {
          this.passwordError.set('Falsches Passwort');
        } else {
          this.passwordError.set('Fehler bei der Überprüfung');
        }
      }
    });
  }

  goBack(): void {
    this.router.navigate(['/timeline']);
  }

  loadSettings(): void {
    this.loading.set(true);
    this.error.set(null);

    this.settingsService.getSettings().subscribe({
      next: (settings) => {
        this.creditIntervalMinutes = settings.credit_interval_minutes;
        this.creditMax = settings.credit_max;
        this.votingPaused.set(settings.voting_paused);
        this.voteVisibilityMode.set(settings.vote_visibility_mode || 'user_choice');
        this.originalCreditIntervalMinutes = settings.credit_interval_minutes;
        this.originalCreditMax = settings.credit_max;
        this.loading.set(false);

        // Load player management data
        this.loadAllUsers();
        this.loadBannedUsers();
      },
      error: (err) => {
        console.error('Failed to load settings:', err);
        this.error.set('Einstellungen konnten nicht geladen werden.');
        this.loading.set(false);
      }
    });
  }

  saveSettings(): void {
    this.saving.set(true);

    this.settingsService.updateSettings({
      credit_interval_minutes: this.creditIntervalMinutes,
      credit_max: this.creditMax
    }).subscribe({
      next: (settings) => {
        this.originalCreditIntervalMinutes = settings.credit_interval_minutes;
        this.originalCreditMax = settings.credit_max;
        this.saving.set(false);
        this.notifications.success('✅ Gespeichert', 'Einstellungen wurden gespeichert und an alle Spieler übertragen');
      },
      error: (err) => {
        console.error('Failed to save settings:', err);
        this.saving.set(false);
        this.notifications.error('❌ Fehler', 'Einstellungen konnten nicht gespeichert werden');
      }
    });
  }

  resetToOriginal(): void {
    this.creditIntervalMinutes = this.originalCreditIntervalMinutes;
    this.creditMax = this.originalCreditMax;
  }

  hasChanges(): boolean {
    return this.creditIntervalMinutes !== this.originalCreditIntervalMinutes ||
           this.creditMax !== this.originalCreditMax;
  }

  resetAllCredits(): void {
    if (!confirm('Bist du sicher? Alle Credits aller Spieler werden auf 0 gesetzt.')) {
      return;
    }

    this.resettingCredits.set(true);
    this.settingsService.resetAllCredits().subscribe({
      next: (response) => {
        this.resettingCredits.set(false);
        this.notifications.success('🔄 Credits zurückgesetzt', `${response.users_affected} Spieler betroffen`);
      },
      error: (err) => {
        console.error('Failed to reset credits:', err);
        this.resettingCredits.set(false);
        this.notifications.error('❌ Fehler', 'Credits konnten nicht zurückgesetzt werden');
      }
    });
  }

  giveEveryoneCredit(): void {
    this.givingCredits.set(true);
    this.settingsService.giveEveryoneCredit().subscribe({
      next: (response) => {
        this.givingCredits.set(false);
        this.notifications.success('🎁 Credit verteilt', `${response.users_affected} Spieler haben 1 Credit erhalten`);
      },
      error: (err) => {
        console.error('Failed to give credits:', err);
        this.givingCredits.set(false);
        this.notifications.error('❌ Fehler', 'Credits konnten nicht verteilt werden');
      }
    });
  }

  toggleVotingPause(): void {
    const newPausedState = !this.votingPaused();
    this.togglingPause.set(true);

    this.settingsService.updateSettings({ voting_paused: newPausedState }).subscribe({
      next: (settings) => {
        this.votingPaused.set(settings.voting_paused);
        this.togglingPause.set(false);
        if (settings.voting_paused) {
          this.notifications.success('⏸️ Voting pausiert', 'Niemand kann jetzt Bewertungen abgeben');
        } else {
          this.notifications.success('▶️ Voting fortgesetzt', 'Bewertungen sind wieder möglich');
        }
      },
      error: (err) => {
        console.error('Failed to toggle voting pause:', err);
        this.togglingPause.set(false);
        this.notifications.error('❌ Fehler', 'Status konnte nicht geändert werden');
      }
    });
  }

  setVoteVisibilityMode(mode: 'user_choice' | 'all_secret' | 'all_public'): void {
    if (this.voteVisibilityMode() === mode) return;

    this.updatingVisibility.set(true);

    this.settingsService.updateSettings({ vote_visibility_mode: mode }).subscribe({
      next: (settings) => {
        this.voteVisibilityMode.set(settings.vote_visibility_mode || 'user_choice');
        this.updatingVisibility.set(false);

        const modeLabels: Record<string, string> = {
          'all_secret': 'Alles geheim',
          'user_choice': 'Nutzer-Wahl',
          'all_public': 'Alles offen'
        };
        this.notifications.success('👁️ Sichtbarkeit geändert', `Modus: ${modeLabels[mode]}`);
      },
      error: (err) => {
        console.error('Failed to update visibility mode:', err);
        this.updatingVisibility.set(false);
        this.notifications.error('❌ Fehler', 'Sichtbarkeit konnte nicht geändert werden');
      }
    });
  }

  invalidateSteamCache(): void {
    this.invalidatingCache.set(true);

    this.gameService.invalidateCache().subscribe({
      next: () => {
        this.invalidatingCache.set(false);
        this.notifications.success('☁️ Steam Cache invalidiert', 'Spiele-Daten werden beim nächsten Laden neu von Steam abgerufen');
      },
      error: (err) => {
        console.error('Failed to invalidate Steam cache:', err);
        this.invalidatingCache.set(false);
        this.notifications.error('❌ Fehler', 'Steam Cache konnte nicht invalidiert werden');
      }
    });
  }

  startDeleteVotesConfirmation(): void {
    this.confirmingDeleteVotes.set(true);
  }

  cancelDeleteVotes(): void {
    this.confirmingDeleteVotes.set(false);
  }

  confirmDeleteAllVotes(): void {
    this.deletingVotes.set(true);

    this.settingsService.deleteAllVotes().subscribe({
      next: (response) => {
        this.deletingVotes.set(false);
        this.confirmingDeleteVotes.set(false);
        this.notifications.success('🗑️ Votes gelöscht', `${response.votes_deleted} Votes wurden gelöscht`);
      },
      error: (err) => {
        console.error('Failed to delete all votes:', err);
        this.deletingVotes.set(false);
        this.notifications.error('❌ Fehler', 'Votes konnten nicht gelöscht werden');
      }
    });
  }

  // Player management methods
  loadAllUsers(): void {
    this.loadingUsers.set(true);
    this.settingsService.getAllUsers().subscribe({
      next: (response) => {
        this.allUsers.set(response.users);
        this.loadingUsers.set(false);
      },
      error: (err) => {
        console.error('Failed to load users:', err);
        this.loadingUsers.set(false);
        this.notifications.error('❌ Fehler', 'Spieler konnten nicht geladen werden');
      }
    });
  }

  loadBannedUsers(): void {
    this.loadingBannedUsers.set(true);
    this.settingsService.getBannedUsers().subscribe({
      next: (response) => {
        this.bannedUsers.set(response.banned_users || []);
        this.loadingBannedUsers.set(false);
      },
      error: (err) => {
        console.error('Failed to load banned users:', err);
        this.loadingBannedUsers.set(false);
        this.notifications.error('❌ Fehler', 'Gebannte Spieler konnten nicht geladen werden');
      }
    });
  }

  startKickUser(user: AdminUserInfo): void {
    this.confirmingAction.set({ userId: user.id, username: user.username, action: 'kick' });
  }

  startBanUser(user: AdminUserInfo): void {
    this.confirmingAction.set({ userId: user.id, username: user.username, action: 'ban' });
  }

  cancelUserAction(): void {
    this.confirmingAction.set(null);
  }

  executeUserAction(): void {
    const action = this.confirmingAction();
    if (!action) return;

    const currentUser = this.authService.user();

    // Admins können sich nicht selbst bannen
    if (action.action === 'ban' && currentUser && currentUser.id === action.userId) {
      this.notifications.error('❌ Nicht erlaubt', 'Du kannst dich nicht selbst bannen');
      this.confirmingAction.set(null);
      return;
    }

    this.executingAction.set(true);

    if (action.action === 'kick') {
      this.settingsService.kickUser(action.userId, 'Kicked by Admin').subscribe({
        next: () => {
          this.executingAction.set(false);
          this.confirmingAction.set(null);
          this.notifications.success('👢 Spieler gekickt', `${action.username} wurde gekickt`);
          this.loadAllUsers();

          // If admin kicked themselves, redirect to login
          if (currentUser && currentUser.id === action.userId) {
            this.authService.logout();
          }
        },
        error: (err) => {
          console.error('Failed to kick user:', err);
          this.executingAction.set(false);
          this.notifications.error('❌ Fehler', 'Spieler konnte nicht gekickt werden');
        }
      });
    } else {
      this.settingsService.banUser(action.userId, 'Banned by Admin').subscribe({
        next: () => {
          this.executingAction.set(false);
          this.confirmingAction.set(null);
          this.notifications.success('🔨 Spieler gebannt', `${action.username} wurde gebannt`);
          this.loadAllUsers();
          this.loadBannedUsers();
        },
        error: (err) => {
          console.error('Failed to ban user:', err);
          this.executingAction.set(false);
          this.notifications.error('❌ Fehler', 'Spieler konnte nicht gebannt werden');
        }
      });
    }
  }

  unbanUser(banned: BannedUser): void {
    this.executingAction.set(true);
    this.settingsService.unbanUser(banned.steam_id).subscribe({
      next: () => {
        this.executingAction.set(false);
        this.notifications.success('✅ Spieler entbannt', `${banned.username} wurde entbannt`);
        this.loadBannedUsers();
      },
      error: (err) => {
        console.error('Failed to unban user:', err);
        this.executingAction.set(false);
        this.notifications.error('❌ Fehler', 'Spieler konnte nicht entbannt werden');
      }
    });
  }

  isCurrentUser(user: AdminUserInfo): boolean {
    const currentUser = this.authService.user();
    return currentUser !== null && currentUser.id === user.id;
  }
}
