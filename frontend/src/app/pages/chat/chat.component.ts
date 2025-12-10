import { Component, OnInit, OnDestroy, inject, signal, ViewChild, ElementRef, AfterViewChecked, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ChatService } from '../../services/chat.service';
import { AuthService } from '../../services/auth.service';
import { AchievementService } from '../../services/achievement.service';
import { ChatMessage, AchievementBadge } from '../../models/chat.model';

@Component({
  selector: 'app-chat',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="page chat-page fade-in">
      <div class="page-header">
        <h1 class="page-title">💬 Chat</h1>
        <p class="page-subtitle">Tausche dich mit anderen Spielern aus</p>
      </div>

      <div class="chat-container">
        <div class="chat-messages" #messagesContainer>
          @if (loading()) {
            <div class="loading-container">
              <div class="spinner"></div>
            </div>
          } @else if (messages().length === 0) {
            <div class="empty-state">
              <span class="empty-state-icon">💬</span>
              <p class="empty-state-title">Noch keine Nachrichten</p>
              <p>Sei der Erste, der etwas schreibt!</p>
            </div>
          } @else {
            @for (msg of messages(); track msg.id) {
              <div class="chat-message" [class.own]="isOwnMessage(msg)">
                <img
                  [src]="msg.user.avatar_small || msg.user.avatar_url || '/assets/default-avatar.png'"
                  [alt]="msg.user.username"
                  class="avatar"
                />
                <div class="message-content">
                  <div class="message-header">
                    <span class="username">{{ msg.user.username }}</span>
                    @if (msg.achievements && msg.achievements.length > 0) {
                      <div class="achievement-badges">
                        @for (badge of msg.achievements; track badge.id) {
                          <span
                            class="badge-wrapper"
                          >
                            <span
                              class="badge"
                              [class.positive]="badge.is_positive"
                              [class.negative]="!badge.is_positive"
                            >
                              <span class="shine-overlay"></span>
                              @if (badge.image_url) {
                                <img [src]="badge.image_url" [alt]="badge.name" class="badge-icon" />
                              }
                            </span>
                            <div class="badge-tooltip">
                              <div class="tooltip-name">{{ badge.name }}</div>
                              @if (getAchievementDescription(badge.id)) {
                                <div class="tooltip-description">{{ getAchievementDescription(badge.id) }}</div>
                              }
                            </div>
                          </span>
                        }
                      </div>
                    }
                    <span class="timestamp">{{ formatTime(msg.created_at) }}</span>
                  </div>
                  <div class="message-text">{{ msg.message }}</div>
                </div>
              </div>
            }
          }
        </div>

        <form class="chat-input" (ngSubmit)="sendMessage()">
          <input
            #messageInput
            type="text"
            [(ngModel)]="newMessage"
            name="message"
            placeholder="Nachricht schreiben..."
            [disabled]="sending()"
            maxlength="500"
            autocomplete="off"
          />
          <button type="submit" [disabled]="!canSend()" class="btn btn-primary">
            @if (sending()) {
              <span class="spinner-small"></span>
            } @else {
              Senden
            }
          </button>
        </form>
      </div>
    </div>
  `,
  styles: [`
    @use 'variables' as *;

    .chat-page {
      height: calc(100vh - 120px);
      display: flex;
      flex-direction: column;
    }

    .chat-container {
      flex: 1;
      display: flex;
      flex-direction: column;
      background: $bg-card;
      border: 1px solid $border-color;
      border-radius: $radius-lg;
      overflow: hidden;
      min-height: 0;
    }

    .chat-messages {
      flex: 1;
      overflow-y: auto;
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .loading-container {
      display: flex;
      justify-content: center;
      align-items: center;
      height: 100%;
    }

    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: $text-muted;
      text-align: center;

      .empty-state-icon {
        font-size: 48px;
        margin-bottom: 16px;
      }

      .empty-state-title {
        font-size: 18px;
        font-weight: 600;
        color: $text-secondary;
        margin-bottom: 8px;
      }
    }

    .chat-message {
      display: flex;
      gap: 12px;
      max-width: 80%;

      &.own {
        align-self: flex-end;
        flex-direction: row-reverse;

        .message-content {
          background: rgba($accent-primary, 0.15);
          border-color: $accent-primary;
        }

        .message-header {
          flex-direction: row-reverse;
        }
      }

      .avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        flex-shrink: 0;
      }

      .message-content {
        background: $bg-hover;
        border: 1px solid $border-color;
        border-radius: $radius-md;
        padding: 10px 14px;
        min-width: 0;
      }

      .message-header {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 8px;
        margin-bottom: 4px;

        .username {
          font-weight: 600;
          color: $text-primary;
          font-size: 14px;
        }

        .timestamp {
          font-size: 12px;
          color: $text-muted;
          margin-left: auto;
        }
      }

      .message-text {
        color: $text-secondary;
        word-break: break-word;
        line-height: 1.4;
      }
    }

    .achievement-badges {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;

      .badge-wrapper {
        position: relative;
        display: inline-flex;

        &:hover .badge-tooltip {
          display: block;
        }
      }

      .badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 4px 10px;
        border-radius: $radius-md;
        font-size: 13px;
        font-weight: 600;
        position: relative;
        overflow: hidden;

        &.positive {
          background: linear-gradient(135deg, #b8860b 0%, #ffd700 50%, #b8860b 100%);
          color: #1a1a1a;
          border: 1px solid #ffd700;
          box-shadow: 0 0 8px rgba(255, 215, 0, 0.4);
          text-shadow: 0 1px 1px rgba(255, 255, 255, 0.3);

          .shine-overlay {
            position: absolute;
            top: 0;
            left: -100%;
            width: 100%;
            height: 100%;
            background: linear-gradient(
              90deg,
              transparent,
              rgba(255, 255, 255, 0.4),
              transparent
            );
            animation: shine 2s infinite;
          }

          .badge-icon {
            filter: drop-shadow(0 0 2px rgba(255, 215, 0, 0.6));
          }
        }

        &.negative {
          background: rgba($accent-negative, 0.2);
          color: $accent-negative;

          .shine-overlay {
            display: none;
          }
        }

        .badge-icon {
          width: 20px;
          height: 20px;
          position: relative;
          z-index: 1;
        }
      }

      .badge-tooltip {
        display: none;
        position: absolute;
        bottom: calc(100% + 8px);
        left: 50%;
        transform: translateX(-50%);
        background: $bg-card;
        border: 1px solid $border-light;
        border-radius: $radius-md;
        padding: 10px 14px;
        min-width: 180px;
        max-width: 250px;
        box-shadow: $shadow-lg;
        z-index: 1000;
        text-align: center;
        pointer-events: none;

        &::after {
          content: '';
          position: absolute;
          top: 100%;
          left: 50%;
          transform: translateX(-50%);
          border: 6px solid transparent;
          border-top-color: $border-light;
        }

        .tooltip-name {
          font-size: 14px;
          font-weight: 700;
          color: $text-primary;
          margin-bottom: 4px;
        }

        .tooltip-description {
          font-size: 12px;
          color: $text-secondary;
          line-height: 1.4;
        }
      }
    }

    @keyframes shine {
      0% {
        left: -100%;
      }
      50%, 100% {
        left: 100%;
      }
    }

    .chat-input {
      display: flex;
      gap: 12px;
      padding: 16px;
      background: $bg-hover;
      border-top: 1px solid $border-color;

      input {
        flex: 1;
        padding: 12px 16px;
        background: $bg-card;
        border: 1px solid $border-color;
        border-radius: $radius-md;
        color: $text-primary;
        font-size: 14px;
        outline: none;
        transition: border-color $transition-base;

        &:focus {
          border-color: $accent-primary;
        }

        &::placeholder {
          color: $text-muted;
        }

        &:disabled {
          opacity: 0.6;
        }
      }

      .btn {
        padding: 12px 24px;
        flex-shrink: 0;
      }
    }

    .spinner-small {
      width: 16px;
      height: 16px;
      border: 2px solid rgba(255, 255, 255, 0.3);
      border-top-color: white;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to {
        transform: rotate(360deg);
      }
    }
  `]
})
export class ChatComponent implements OnInit, OnDestroy, AfterViewChecked {
  private chatService = inject(ChatService);
  private authService = inject(AuthService);
  private achievementService = inject(AchievementService);

  @ViewChild('messagesContainer') private messagesContainer!: ElementRef;
  @ViewChild('messageInput') private messageInput!: ElementRef<HTMLInputElement>;

  messages = this.chatService.chatMessages;
  loading = signal(true);
  sending = signal(false);
  newMessage = '';

  private shouldScrollToBottom = false;

  constructor() {
    // React to new messages being added
    effect(() => {
      const msgs = this.messages();
      if (msgs.length > 0) {
        this.shouldScrollToBottom = true;
      }
    });
  }

  ngOnInit(): void {
    this.chatService.setChatOpen(true);
    this.loadMessages();
  }

  ngOnDestroy(): void {
    this.chatService.setChatOpen(false);
  }

  ngAfterViewChecked(): void {
    if (this.shouldScrollToBottom) {
      this.scrollToBottom();
      this.shouldScrollToBottom = false;
    }
  }

  private loadMessages(): void {
    this.loading.set(true);
    this.chatService.loadMessages().subscribe({
      next: () => {
        this.loading.set(false);
        this.shouldScrollToBottom = true;
      },
      error: (err) => {
        console.error('Failed to load chat messages', err);
        this.loading.set(false);
      }
    });
  }

  sendMessage(): void {
    const message = this.newMessage.trim();
    if (!message || this.sending()) return;

    this.sending.set(true);
    this.chatService.sendMessage(message).subscribe({
      next: () => {
        this.newMessage = '';
        this.sending.set(false);
        this.shouldScrollToBottom = true;
        // Keep focus on input field
        setTimeout(() => this.messageInput?.nativeElement?.focus(), 0);
      },
      error: (err) => {
        console.error('Failed to send message', err);
        this.sending.set(false);
        // Keep focus on input field even on error
        setTimeout(() => this.messageInput?.nativeElement?.focus(), 0);
      }
    });
  }

  canSend(): boolean {
    return this.newMessage.trim().length > 0 && !this.sending();
  }

  isOwnMessage(msg: ChatMessage): boolean {
    const currentUser = this.authService.user();
    return currentUser?.id === msg.user.id;
  }

  formatTime(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);

    if (diffMins < 1) return 'gerade eben';
    if (diffMins < 60) return `vor ${diffMins} Min`;
    if (diffHours < 24) return `vor ${diffHours} Std`;

    return date.toLocaleDateString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  private scrollToBottom(): void {
    if (this.messagesContainer) {
      const el = this.messagesContainer.nativeElement;
      el.scrollTop = el.scrollHeight;
    }
  }

  getAchievementDescription(id: string): string {
    return this.achievementService.getDescription(id);
  }
}
