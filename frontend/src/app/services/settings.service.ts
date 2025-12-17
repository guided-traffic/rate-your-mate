import { Injectable, signal, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { environment } from '../../environments/environment';
import { Settings, UpdateSettingsRequest, CreditActionResponse } from '../models/settings.model';

export interface VotingStatusResponse {
  voting_paused: boolean;
  negative_voting_disabled: boolean;
  countdown_target?: string | null; // RFC3339 formatted time, null if not set
}

export interface AdminPasswordRequiredResponse {
  password_required: boolean;
}

export interface VerifyAdminPasswordResponse {
  valid: boolean;
  password_required: boolean;
  error?: string;
}

export interface AdminUserInfo {
  id: number;
  steam_id: string;
  username: string;
  avatar_small: string;
  created_at: string;
}

export interface BannedUser {
  id: number;
  steam_id: string;
  username: string;
  reason: string;
  banned_by: string;
  banned_at: string;
}

export interface KickBanResponse {
  message: string;
  username: string;
}

@Injectable({
  providedIn: 'root'
})
export class SettingsService {
  private settings = signal<Settings | null>(null);
  readonly currentSettings = this.settings.asReadonly();

  // Global voting paused state (can be set via WebSocket before admin loads settings)
  private votingPausedSignal = signal(false);
  readonly votingPaused = this.votingPausedSignal.asReadonly();

  // Global negative voting disabled state
  private negativeVotingDisabledSignal = signal(false);
  readonly negativeVotingDisabled = this.negativeVotingDisabledSignal.asReadonly();

  // Countdown target (RFC3339 formatted time, null if not set)
  private countdownTargetSignal = signal<string | null>(null);
  readonly countdownTarget = this.countdownTargetSignal.asReadonly();

  constructor(private http: HttpClient) {
    // Load voting status on service init
    this.loadVotingStatus();
  }

  // Load voting status (accessible to all authenticated users)
  loadVotingStatus(): void {
    this.http.get<VotingStatusResponse>(`${environment.apiUrl}/voting-status`).subscribe({
      next: (response) => {
        this.votingPausedSignal.set(response.voting_paused);
        this.negativeVotingDisabledSignal.set(response.negative_voting_disabled);
        this.countdownTargetSignal.set(response.countdown_target || null);
      },
      error: (err) => {
        console.error('Failed to load voting status:', err);
      }
    });
  }

  // Load countdown for public access (no auth required) - returns Observable
  getPublicCountdown(): Observable<{ countdown_target?: string | null }> {
    return this.http.get<{ countdown_target?: string | null }>(`${environment.apiUrl}/countdown`);
  }

  // Check if admin password is required
  checkAdminPasswordRequired(): Observable<AdminPasswordRequiredResponse> {
    return this.http.get<AdminPasswordRequiredResponse>(`${environment.apiUrl}/admin/password-required`);
  }

  // Verify admin password
  verifyAdminPassword(password: string): Observable<VerifyAdminPasswordResponse> {
    return this.http.post<VerifyAdminPasswordResponse>(`${environment.apiUrl}/admin/verify-password`, { password });
  }

  getSettings(): Observable<Settings> {
    return this.http.get<Settings>(`${environment.apiUrl}/admin/settings`).pipe(
      tap(settings => {
        this.settings.set(settings);
        this.votingPausedSignal.set(settings.voting_paused);
        this.negativeVotingDisabledSignal.set(settings.negative_voting_disabled);
        this.countdownTargetSignal.set(settings.countdown_target || null);
      })
    );
  }

  updateSettings(request: UpdateSettingsRequest): Observable<Settings> {
    return this.http.put<Settings>(`${environment.apiUrl}/admin/settings`, request).pipe(
      tap(settings => {
        this.settings.set(settings);
        this.votingPausedSignal.set(settings.voting_paused);
        this.negativeVotingDisabledSignal.set(settings.negative_voting_disabled);
        this.countdownTargetSignal.set(settings.countdown_target || null);
      })
    );
  }

  resetAllCredits(): Observable<CreditActionResponse> {
    return this.http.post<CreditActionResponse>(`${environment.apiUrl}/admin/credits/reset`, {});
  }

  giveEveryoneCredit(): Observable<CreditActionResponse> {
    return this.http.post<CreditActionResponse>(`${environment.apiUrl}/admin/credits/give`, {});
  }

  deleteAllVotes(): Observable<{ message: string; votes_deleted: number }> {
    return this.http.post<{ message: string; votes_deleted: number }>(`${environment.apiUrl}/admin/votes/delete-all`, {});
  }

  // User management
  getAllUsers(): Observable<{ users: AdminUserInfo[] }> {
    return this.http.get<{ users: AdminUserInfo[] }>(`${environment.apiUrl}/admin/users`);
  }

  getBannedUsers(): Observable<{ banned_users: BannedUser[] }> {
    return this.http.get<{ banned_users: BannedUser[] }>(`${environment.apiUrl}/admin/users/banned`);
  }

  kickUser(userId: number, reason?: string): Observable<KickBanResponse> {
    return this.http.post<KickBanResponse>(`${environment.apiUrl}/admin/users/${userId}/kick`, { reason });
  }

  banUser(userId: number, reason?: string): Observable<KickBanResponse> {
    return this.http.post<KickBanResponse>(`${environment.apiUrl}/admin/users/${userId}/ban`, { reason });
  }

  unbanUser(steamId: string): Observable<KickBanResponse> {
    return this.http.post<KickBanResponse>(`${environment.apiUrl}/admin/users/unban/${steamId}`, {});
  }

  // Called by WebSocket service when settings are updated
  applySettingsUpdate(settings: Partial<Settings>): void {
    if (settings.voting_paused !== undefined) {
      this.votingPausedSignal.set(settings.voting_paused);
    }
    if (settings.negative_voting_disabled !== undefined) {
      this.negativeVotingDisabledSignal.set(settings.negative_voting_disabled);
    }
    if (settings.countdown_target !== undefined) {
      this.countdownTargetSignal.set(settings.countdown_target || null);
    }
    const current = this.settings();
    if (current) {
      this.settings.set({ ...current, ...settings });
    }
  }
}
