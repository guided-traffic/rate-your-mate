import { Injectable, signal, inject } from '@angular/core';
import { environment } from '../../environments/environment';
import { AuthService } from './auth.service';
import { ConnectionStatusService } from './connection-status.service';
import { WebSocketMessage, VotePayload, SettingsPayload, CreditActionPayload, ChatMessagePayload, NewKingPayload, GamesSyncProgressPayload, GamesSyncCompletePayload, VoteInvalidationPayload } from '../models/websocket.model';
import { Subject, Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class WebSocketService {
  private authService = inject(AuthService);
  private connectionStatus = inject(ConnectionStatusService);

  private socket: WebSocket | null = null;
  private wasConnected = false; // Track if we were ever connected

  private connected = signal(false);
  readonly isConnected = this.connected.asReadonly();

  // Subjects for different message types
  readonly voteReceived$ = new Subject<VotePayload>();
  readonly newVote$ = new Subject<VotePayload>();
  readonly settingsUpdate$ = new Subject<SettingsPayload>();
  readonly creditsReset$ = new Subject<CreditActionPayload>();
  readonly creditsGiven$ = new Subject<CreditActionPayload>();
  readonly chatMessage$ = new Subject<ChatMessagePayload>();
  readonly newKing$ = new Subject<NewKingPayload>();
  readonly gamesSyncProgress$ = new Subject<GamesSyncProgressPayload>();
  readonly gamesSyncComplete$ = new Subject<GamesSyncCompletePayload>();
  readonly voteInvalidation$ = new Subject<VoteInvalidationPayload>();

  // General messages observable for timeline component
  private messagesSubject = new Subject<{ type: string; payload: VotePayload }>();
  readonly messages$: Observable<{ type: string; payload: VotePayload }> = this.messagesSubject.asObservable();

  connect(): void {
    const token = this.authService.getToken();
    if (!token) {
      console.warn('WebSocket: No token available');
      return;
    }

    // Check if already connected or connecting
    if (this.socket) {
      if (this.socket.readyState === WebSocket.OPEN) {
        console.log('WebSocket: Already connected');
        return;
      }
      if (this.socket.readyState === WebSocket.CONNECTING) {
        console.log('WebSocket: Connection in progress');
        return;
      }
      // Close any existing socket that's closing or closed
      this.socket = null;
    }

    const wsUrl = `${environment.wsUrl}?token=${token}`;
    console.log('WebSocket: Connecting to', wsUrl);

    try {
      this.socket = new WebSocket(wsUrl);

      this.socket.onopen = () => {
        console.log('WebSocket: Connected successfully');
        this.connected.set(true);
        this.wasConnected = true;
      };

      this.socket.onmessage = (event) => {
        try {
          const message: WebSocketMessage<VotePayload> = JSON.parse(event.data);
          console.log('WebSocket: Received message', message.type);
          this.handleMessage(message);
        } catch (error) {
          console.error('WebSocket: Failed to parse message', error, event.data);
        }
      };

      this.socket.onclose = (event) => {
        console.log('WebSocket: Disconnected', event.code, event.reason);
        this.connected.set(false);
        this.socket = null;

        // Only show reconnect spinner if we were previously connected
        // This prevents showing spinner on initial connection failures
        if (event.code !== 1000 && this.wasConnected && this.authService.isAuthenticated()) {
          // Show spinner and start reconnect via ConnectionStatusService
          // This will reload the page when backend is back
          this.connectionStatus.setDisconnected();
        }
      };

      this.socket.onerror = (error) => {
        console.error('WebSocket: Error', error);
        // Note: onclose will be called after onerror, so we don't set disconnected here
      };
    } catch (error) {
      console.error('WebSocket: Failed to create connection', error);
      this.socket = null;
    }
  }

  disconnect(): void {
    if (this.socket) {
      console.log('WebSocket: Disconnecting...');
      this.socket.close(1000, 'User logout');
      this.socket = null;
      this.connected.set(false);
    }
  }

  private handleMessage(message: WebSocketMessage<VotePayload | SettingsPayload | CreditActionPayload | ChatMessagePayload | NewKingPayload | GamesSyncProgressPayload | GamesSyncCompletePayload | VoteInvalidationPayload>): void {
    switch (message.type) {
      case 'new_vote':
        console.log('WebSocket: New vote received', message.payload);
        this.newVote$.next(message.payload as VotePayload);
        this.messagesSubject.next({ type: 'new_vote', payload: message.payload as VotePayload });
        break;
      case 'settings_update':
        console.log('WebSocket: Settings update received', message.payload);
        this.settingsUpdate$.next(message.payload as SettingsPayload);
        break;
      case 'credits_reset':
        console.log('WebSocket: Credits reset received', message.payload);
        this.creditsReset$.next(message.payload as CreditActionPayload);
        break;
      case 'credits_given':
        console.log('WebSocket: Credits given received', message.payload);
        this.creditsGiven$.next(message.payload as CreditActionPayload);
        break;
      case 'chat_message':
        console.log('WebSocket: Chat message received', message.payload);
        this.chatMessage$.next(message.payload as ChatMessagePayload);
        break;
      case 'new_king':
        console.log('WebSocket: New king received', message.payload);
        this.newKing$.next(message.payload as NewKingPayload);
        break;
      case 'games_sync_progress':
        console.log('WebSocket: Games sync progress received', message.payload);
        this.gamesSyncProgress$.next(message.payload as GamesSyncProgressPayload);
        break;
      case 'games_sync_complete':
        console.log('WebSocket: Games sync complete received', message.payload);
        this.gamesSyncComplete$.next(message.payload as GamesSyncCompletePayload);
        break;
      case 'vote_invalidation':
        console.log('WebSocket: Vote invalidation received', message.payload);
        this.voteInvalidation$.next(message.payload as VoteInvalidationPayload);
        break;
      default:
        console.log('WebSocket: Unknown message type', message.type);
    }
  }
}
