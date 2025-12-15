import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { VoteService } from '../../services/vote.service';
import { WebSocketService } from '../../services/websocket.service';
import { AuthService } from '../../services/auth.service';
import { Vote } from '../../models/vote.model';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-timeline',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="page fade-in">
      <div class="page-header">
        <h1 class="page-title">Live Timeline</h1>
        <p class="page-subtitle">Alle Achievements in Echtzeit</p>
      </div>

      @if (loading()) {
        <div class="loading-container">
          <div class="spinner"></div>
        </div>
      } @else if (votes().length === 0) {
        <div class="empty-state">
          <span class="empty-state-icon">🕐</span>
          <p class="empty-state-title">Noch keine Achievements vergeben</p>
          <p>Sei der Erste, der einen anderen Spieler bewertet!</p>
        </div>
      } @else {
        <div class="timeline">
          @for (vote of votes(); track vote.id) {
            <div
              class="timeline-item"
              [class.positive]="vote.achievement.is_positive !== false"
              [class.negative]="vote.achievement.is_positive === false"
              [class.new]="isNew(vote)"
              [class.invalidated]="vote.is_invalidated"
            >
              <div class="timeline-avatars">
                <div class="avatar-wrapper">
                  <img
                    [src]="vote.from_user.avatar_small || '/assets/default-avatar.png'"
                    [alt]="vote.from_user.username"
                    class="avatar"
                  />
                  <span class="avatar-name">{{ vote.from_user.username }}</span>
                </div>
                <span class="arrow">→</span>
                <div class="avatar-wrapper">
                  <img
                    [src]="vote.to_user.avatar_small || '/assets/default-avatar.png'"
                    [alt]="vote.to_user.username"
                    class="avatar"
                  />
                  <span class="avatar-name">{{ vote.to_user.username }}</span>
                </div>
              </div>

              <div class="timeline-content">
                <div class="timeline-text">
                  <span
                    class="achievement-chip"
                    [class.positive]="vote.achievement.is_positive !== false"
                    [class.negative]="vote.achievement.is_positive === false"
                  >
                    @if (vote.achievement.image_url) {
                      <img [src]="vote.achievement.image_url" [alt]="vote.achievement.name" class="achievement-icon" />
                    }
                    <span class="achievement-info">
                      <span class="achievement-name">{{ vote.achievement.name || vote.achievement_id }}</span>
                      <span class="points-badge">+{{ vote.points }}</span>
                    </span>
                  </span>
                  @if (vote.comment) {
                    <span class="comment-bubble">{{ vote.comment }}</span>
                  }
                </div>
              </div>

              <div class="timeline-meta">
                <span class="timeline-time">
                  <span class="time-absolute">{{ formatAbsoluteTime(vote.created_at) }}</span>
                  <span class="time-relative">({{ formatRelativeTime(vote.created_at) }})</span>
                </span>
              </div>

              @if (isAdmin()) {
                <button
                  class="invalidate-btn"
                  [class.active]="vote.is_invalidated"
                  (click)="toggleInvalidation(vote)"
                  [disabled]="invalidatingVoteId() === vote.id"
                  [title]="vote.is_invalidated ? 'Vote wieder aktivieren' : 'Vote invalidieren'"
                >
                  @if (invalidatingVoteId() === vote.id) {
                    <span class="spinner-small"></span>
                  } @else {
                    {{ vote.is_invalidated ? '✓' : '✕' }}
                  }
                </button>
              }
            </div>
          }
        </div>
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

    .timeline {
      display: flex;
      flex-direction: column;
      gap: 16px;
    }

    .timeline-item {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 16px 20px;
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      transition: all $transition-base;
      position: relative;

      &:hover {
        border-color: $border-light;
      }

      &.positive {
        border-left: 3px solid $accent-positive;
      }

      &.negative {
        border-left: 3px solid $accent-negative;
      }

      &.new {
        animation: slideIn 0.3s ease-out;
        background: rgba($accent-primary, 0.05);
        border-color: $accent-primary;
      }

      &.invalidated {
        opacity: 0.35;
        filter: grayscale(50%);

        &:hover {
          opacity: 0.5;
        }
      }
    }

    .invalidate-btn {
      position: absolute;
      bottom: 8px;
      right: 12px;
      display: flex;
      align-items: center;
      justify-content: center;
      width: 28px;
      height: 28px;
      border: 1px solid $border-color;
      border-radius: $radius-sm;
      background: $bg-tertiary;
      color: $text-muted;
      font-size: 14px;
      cursor: pointer;
      transition: all $transition-base;

      &:hover {
        background: $bg-card;
        border-color: $accent-negative;
        color: $accent-negative;
      }

      &.active {
        background: rgba($accent-positive, 0.15);
        border-color: $accent-positive;
        color: $accent-positive;

        &:hover {
          background: rgba($accent-positive, 0.25);
        }
      }

      &:disabled {
        cursor: not-allowed;
        opacity: 0.5;
      }
    }

    .spinner-small {
      width: 14px;
      height: 14px;
      border: 2px solid $border-color;
      border-top-color: $accent-primary;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    @keyframes slideIn {
      from {
        opacity: 0;
        transform: translateY(-20px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    .timeline-avatars {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-shrink: 0;

      .avatar-wrapper {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 6px;
        width: 130px;
      }

      .avatar {
        position: relative;
        z-index: 1;
        width: 64px;
        height: 64px;
        border-radius: 50%;
      }

      .avatar-name {
        font-size: 13px;
        font-weight: 500;
        color: $text-secondary;
        text-align: center;
        width: 100%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .arrow {
        color: $text-secondary;
        font-size: 32px;
        position: relative;
        z-index: 2;
        margin-bottom: 22px;
        flex-shrink: 0;
      }
    }

    .timeline-content {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 8px;
      min-width: 0;
    }

    .timeline-text {
      display: flex;
      align-items: flex-start;
      gap: 12px;
      font-size: 14px;
      line-height: 1.5;
      flex-wrap: wrap;

      .username {
        font-weight: 600;
        color: $text-primary;
      }
    }

    .comment-bubble {
      position: relative;
      background: $bg-tertiary;
      border: 1px solid $border-color;
      border-radius: 12px;
      padding: 6px 12px;
      font-size: 13px;
      color: $text-secondary;
      line-height: 1.4;
      max-width: 300px;
      word-wrap: break-word;

      // Speech bubble arrow pointing left
      &::before {
        content: '';
        position: absolute;
        top: 50%;
        left: -6px;
        width: 10px;
        height: 10px;
        background: $bg-tertiary;
        border-left: 1px solid $border-color;
        border-bottom: 1px solid $border-color;
        transform: translateY(-50%) rotate(45deg);
      }
    }

    .timeline-meta {
      position: absolute;
      top: 8px;
      right: 12px;
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .timeline-time {
      display: flex;
      gap: 6px;
      font-size: 12px;
      color: $text-muted;
      white-space: nowrap;

      .time-absolute {
        color: $text-secondary;
        font-weight: 500;
      }

      .time-relative {
        color: $text-muted;
      }
    }

    @media (max-width: 600px) {
      .timeline-item {
        flex-direction: column;
        align-items: flex-start;
        gap: 12px;
      }

      .timeline-avatars {
        align-self: center;
      }

      .timeline-content {
        text-align: center;
        align-items: center;
        width: 100%;
      }

      .timeline-meta {
        flex-direction: row;
        justify-content: center;
        align-items: center;
        width: 100%;
        gap: 12px;
      }
    }
  `]
})
export class TimelineComponent implements OnInit, OnDestroy {
  private voteService = inject(VoteService);
  private wsService = inject(WebSocketService);
  private authService = inject(AuthService);

  votes = signal<Vote[]>([]);
  loading = signal(true);
  newVoteIds = signal<Set<number>>(new Set());
  invalidatingVoteId = signal<number | null>(null);

  isAdmin = computed(() => this.authService.user()?.is_admin ?? false);

  private wsSubscription?: Subscription;
  private settingsSubscription?: Subscription;
  private invalidationSubscription?: Subscription;

  ngOnInit(): void {
    this.loadVotes();
    this.subscribeToLiveVotes();
    this.subscribeToSettingsUpdates();
    this.subscribeToVoteInvalidation();
  }

  ngOnDestroy(): void {
    this.wsSubscription?.unsubscribe();
    this.settingsSubscription?.unsubscribe();
    this.invalidationSubscription?.unsubscribe();
  }

  loadVotes(): void {
    this.voteService.getAll().subscribe({
      next: (votes) => {
        this.votes.set(votes);
        this.loading.set(false);
      },
      error: (error) => {
        console.error('Failed to load votes:', error);
        this.loading.set(false);
      }
    });
  }

  subscribeToLiveVotes(): void {
    // Subscribe to new_vote broadcasts - updates timeline for all users
    this.wsSubscription = this.wsService.newVote$.subscribe((payload) => {
      console.log('Timeline: Received new vote', payload.vote_id);
      // Reload the timeline to get full vote details
      this.voteService.getAll().subscribe({
        next: (votes) => {
          const currentVotes = this.votes();
          // Find new votes
          const newVoteIds = votes
            .filter(v => !currentVotes.some(cv => cv.id === v.id))
            .map(v => v.id);

          this.votes.set(votes);

          // Mark new votes for animation
          if (newVoteIds.length > 0) {
            const ids = new Set(this.newVoteIds());
            newVoteIds.forEach(id => ids.add(id));
            this.newVoteIds.set(ids);

            // Remove "new" indicator after animation
            setTimeout(() => {
              const currentIds = new Set(this.newVoteIds());
              newVoteIds.forEach(id => currentIds.delete(id));
              this.newVoteIds.set(currentIds);
            }, 3000);
          }
        }
      });
    });
  }

  subscribeToSettingsUpdates(): void {
    // Subscribe to settings updates - reload votes when visibility mode changes
    this.settingsSubscription = this.wsService.settingsUpdate$.subscribe((settings) => {
      console.log('Timeline: Settings updated, reloading votes');
      // Reload votes to reflect new visibility mode
      this.voteService.getAll().subscribe({
        next: (votes) => {
          this.votes.set(votes);
        }
      });
    });
  }

  subscribeToVoteInvalidation(): void {
    // Subscribe to vote invalidation updates
    this.invalidationSubscription = this.wsService.voteInvalidation$.subscribe((payload) => {
      console.log('Timeline: Vote invalidation update', payload.vote_id, payload.is_invalidated);
      // Update the specific vote's invalidation status
      const currentVotes = this.votes();
      const updatedVotes = currentVotes.map(vote =>
        vote.id === payload.vote_id
          ? { ...vote, is_invalidated: payload.is_invalidated }
          : vote
      );
      this.votes.set(updatedVotes);
    });
  }

  toggleInvalidation(vote: Vote): void {
    this.invalidatingVoteId.set(vote.id);
    this.voteService.toggleInvalidation(vote.id).subscribe({
      next: (response) => {
        // Update handled via WebSocket, but also update locally for immediate feedback
        const currentVotes = this.votes();
        const updatedVotes = currentVotes.map(v =>
          v.id === vote.id
            ? { ...v, is_invalidated: response.is_invalidated }
            : v
        );
        this.votes.set(updatedVotes);
        this.invalidatingVoteId.set(null);
      },
      error: (error) => {
        console.error('Failed to toggle vote invalidation:', error);
        this.invalidatingVoteId.set(null);
      }
    });
  }

  isNew(vote: Vote): boolean {
    return this.newVoteIds().has(vote.id);
  }

  formatAbsoluteTime(dateStr: string | undefined): string {
    if (!dateStr) return '';

    const date = new Date(dateStr);
    return date.toLocaleTimeString('de-DE', {
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  formatRelativeTime(dateStr: string | undefined): string {
    if (!dateStr) return '';

    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'gerade eben';
    if (diffMins < 60) return `vor ${diffMins} Min`;
    if (diffHours < 24) return `vor ${diffHours} Std`;
    if (diffDays < 7) return `vor ${diffDays} Tagen`;

    return date.toLocaleDateString('de-DE', {
      day: '2-digit',
      month: '2-digit'
    });
  }

  getPointsArray(points: number): number[] {
    // Use absolute value for negative points, ensure at least 1 is shown
    const count = Math.abs(points) || 1;
    return Array.from({ length: count }, (_, i) => i);
  }

  getPointsLabel(points: number): string {
    const count = Math.abs(points) || 1;
    return count === 1 ? '1 Credit' : `${count} Credits`;
  }
}
