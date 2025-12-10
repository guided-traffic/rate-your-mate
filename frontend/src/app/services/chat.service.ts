import { Injectable, signal, inject, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../environments/environment';
import { ChatMessage, ChatMessagesResponse, CreateChatMessageRequest } from '../models/chat.model';
import { ChatMessagePayload } from '../models/websocket.model';
import { WebSocketService } from './websocket.service';
import { AuthService } from './auth.service';

@Injectable({
  providedIn: 'root'
})
export class ChatService {
  private http = inject(HttpClient);
  private wsService = inject(WebSocketService);
  private authService = inject(AuthService);

  private messages = signal<ChatMessage[]>([]);
  readonly chatMessages = this.messages.asReadonly();

  private _unreadCount = signal<number>(0);
  readonly unreadCount = this._unreadCount.asReadonly();

  private _isChatOpen = signal<boolean>(false);

  constructor() {
    // Subscribe to WebSocket chat messages
    this.wsService.chatMessage$.subscribe((payload) => {
      this.addMessageFromPayload(payload);
    });
  }

  loadMessages(): Observable<ChatMessage[]> {
    return this.http.get<ChatMessagesResponse>(`${environment.apiUrl}/chat`)
      .pipe(
        map(response => {
          const msgs = response.messages || [];
          this.messages.set(msgs);
          return msgs;
        })
      );
  }

  sendMessage(message: string): Observable<{ message: ChatMessage }> {
    const request: CreateChatMessageRequest = { message };
    return this.http.post<{ message: ChatMessage }>(`${environment.apiUrl}/chat`, request);
  }

  private addMessageFromPayload(payload: ChatMessagePayload): void {
    // Convert payload to ChatMessage format
    const newMessage: ChatMessage = {
      id: payload.id,
      user: {
        id: payload.user_id,
        steam_id: payload.steam_id,
        username: payload.username,
        avatar_url: payload.avatar_small || '',
        avatar_small: payload.avatar_small || '',
        profile_url: ''
      },
      message: payload.message,
      achievements: payload.achievements || [],
      created_at: payload.created_at
    };

    // Add to messages array
    this.messages.update(msgs => [...msgs, newMessage]);

    // Increment unread count if chat is not open and message is not from current user
    const currentUser = this.authService.user();
    if (!this._isChatOpen() && currentUser && payload.user_id !== currentUser.id) {
      this._unreadCount.update(count => count + 1);
    }
  }

  clearMessages(): void {
    this.messages.set([]);
  }

  markAsRead(): void {
    this._unreadCount.set(0);
  }

  setChatOpen(isOpen: boolean): void {
    this._isChatOpen.set(isOpen);
    if (isOpen) {
      this.markAsRead();
    }
  }
}
