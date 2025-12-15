import { Injectable, signal, inject, computed } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../environments/environment';
import { User } from '../models/user.model';

export interface PlayerRanking {
  user: User;
  total_score: number;   // net votes + bonus points
  net_votes: number;     // positive - negative votes
  bonus_points: number;  // bonus from achievement placements
  rank: number;
}

export interface MyRankingResponse {
  rank: PlayerRanking | null;
  total_votes: number;
  min_votes_for_ranking: number;
  ranking_active: boolean;
}

export interface GlobalRankingResponse {
  rankings: PlayerRanking[];
  total_votes: number;
  min_votes_for_ranking: number;
  ranking_active: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class RankingService {
  private http = inject(HttpClient);

  private myRanking = signal<PlayerRanking | null>(null);
  private totalVotes = signal<number>(0);
  private minVotesForRanking = signal<number>(10);
  private rankingActive = signal<boolean>(false);
  private loading = signal<boolean>(false);

  readonly rank = this.myRanking.asReadonly();
  readonly isRankingActive = this.rankingActive.asReadonly();
  readonly isLoading = this.loading.asReadonly();
  readonly votesUntilRanking = computed(() => {
    const min = this.minVotesForRanking();
    const total = this.totalVotes();
    return Math.max(0, min - total);
  });

  /** Display string for the rank badge */
  readonly rankDisplay = computed(() => {
    if (!this.rankingActive()) {
      return null; // Show hourglass
    }
    const ranking = this.myRanking();
    if (!ranking) {
      return null; // Show hourglass
    }
    return ranking.rank;
  });

  loadMyRanking(): void {
    this.loading.set(true);
    this.http.get<MyRankingResponse>(`${environment.apiUrl}/ranking/me`).subscribe({
      next: (response) => {
        this.myRanking.set(response.rank);
        this.totalVotes.set(response.total_votes);
        this.minVotesForRanking.set(response.min_votes_for_ranking);
        this.rankingActive.set(response.ranking_active);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Failed to load ranking:', err);
        this.loading.set(false);
      }
    });
  }

  loadGlobalRanking(): Promise<GlobalRankingResponse> {
    return new Promise((resolve, reject) => {
      this.http.get<GlobalRankingResponse>(`${environment.apiUrl}/ranking`).subscribe({
        next: (response) => {
          this.totalVotes.set(response.total_votes);
          this.minVotesForRanking.set(response.min_votes_for_ranking);
          this.rankingActive.set(response.ranking_active);
          resolve(response);
        },
        error: (err) => {
          console.error('Failed to load global ranking:', err);
          reject(err);
        }
      });
    });
  }

  /** Refresh ranking data (call after a new vote) */
  refresh(): void {
    this.loadMyRanking();
  }
}
