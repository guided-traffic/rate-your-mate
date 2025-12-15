import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { environment } from '../../environments/environment';
import { Vote, CreateVoteRequest, VoteResponse, AchievementLeaderboard, ChampionsResult } from '../models/vote.model';

export interface ToggleInvalidationResponse {
  vote_id: number;
  is_invalidated: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class VoteService {
  constructor(private http: HttpClient) {}

  create(request: CreateVoteRequest): Observable<VoteResponse> {
    return this.http.post<VoteResponse>(`${environment.apiUrl}/votes`, request);
  }

  getAll(): Observable<Vote[]> {
    return this.http.get<{ votes: Vote[] }>(`${environment.apiUrl}/votes`)
      .pipe(map(response => response.votes || []));
  }

  getTimeline(): Observable<Vote[]> {
    return this.getAll();
  }

  getLeaderboard(): Observable<AchievementLeaderboard[]> {
    return this.http.get<{ leaderboard: AchievementLeaderboard[] }>(`${environment.apiUrl}/leaderboard`)
      .pipe(map(response => response.leaderboard || []));
  }

  getChampions(): Observable<ChampionsResult> {
    return this.http.get<{ champions: ChampionsResult }>(`${environment.apiUrl}/champions`)
      .pipe(map(response => response.champions));
  }

  toggleInvalidation(voteId: number): Observable<ToggleInvalidationResponse> {
    return this.http.put<ToggleInvalidationResponse>(`${environment.apiUrl}/admin/votes/${voteId}/invalidate`, {});
  }
}
