const state = {
    players: [],
    teams: [],
    query: '',
    openTeam: null,
    openPlayer: null,
    teamStats: {},
};

const teamList = document.querySelector('#team-list');
const search = document.querySelector('#search');
const status = document.querySelector('#directory-status');
const errorState = document.querySelector('#error-state');
const dataUri = 'https://playerinsights.guysports.co.uk/data/';

const formatNumber = (value) => Number(value || 0).toLocaleString('en-GB', { maximumFractionDigits: 1 });
const playerName = (player) => player.displayName || `${player.firstName} ${player.lastName}`;
const teamName = (team, player) => (team && team.name) || (player && player.contestantName) || 'Unassigned';
const escapeHtml = (value) => String(value || '').replace(/[&<>'"]/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[character]));

const rankingMetrics = {
    goals: 'goals',
    assists: 'assists',
    tackles: 'tackles',
    dribbles: 'dribbles',
    crosses: 'crosses',
    offsides: 'offsides',
    blocks: 'blocks',
    interceptions: 'interceptions',
    'fouls won': 'foulsWon',
    foulswon: 'foulsWon',
    'fouls made': 'foulsMade',
    foulsmade: 'foulsMade',
    'shots on target': 'shotsOnTarget',
    shotsontarget: 'shotsOnTarget',
    'chances created': 'chancesCreated',
    chancescreated: 'chancesCreated',
    'clean sheets': 'cleanSheet',
    cleansheets: 'cleanSheet',
    'clean sheet': 'cleanSheet',
    cleansheet: 'cleanSheet',
    'goals conceded': 'goalsConceded',
    goalsconceded: 'goalsConceded',
    saves: 'saves',
    punches: 'punches',
    claims: 'claims',
    'keeper sweeps': 'keeperSweeps',
    keepersweeps: 'keeperSweeps',
    'total points': 'totalPoints',
    totalpoints: 'totalPoints',
    points: 'totalPoints',
    'bonus points': 'bonusPoints',
    bonuspoints: 'bonusPoints',
    'ppm points': 'ppmPoints',
    ppmpoints: 'ppmPoints',
    'bonus ppm': 'bonusPpm',
    bonusppm: 'bonusPpm',
    average: 'averagePoints',
    'average points': 'averagePoints',
    averagepoints: 'averagePoints',
    'pass completion rate': 'passCompletionRate',
    passcompletionrate: 'passCompletionRate',
    'yellow cards': 'yellowCards',
    yellowcards: 'yellowCards',
    'red cards': 'redCards',
    redcards: 'redCards',
    'own goals': 'ownGoals',
    owngoals: 'ownGoals',
    'penalty misses': 'penaltyMisses',
    penaltymisses: 'penaltyMisses',
    'penalty saves': 'penaltySaves',
    penaltysaves: 'penaltySaves',
    'goals outside area': 'goalsOutsideArea',
    goalsoutsidearea: 'goalsOutsideArea',
    'errors leading to goal': 'errorsLeadingToGoal',
    errorsleadingtogoal: 'errorsLeadingToGoal',
};

function parseRankingQuery(query) {
    const normalized = query.trim().toLowerCase();
    const match = normalized.match(/^(?:(?:get|list)(?:\s+me)?\s+)?(top|bottom)(?:\s+(\d+))?\s+(?:players?\s+(?:for|by|with)\s+)?(.+)$/);
    if (!match) return null;
    const metric = rankingMetrics[match[3].trim()];
    if (!metric) return null;
    return { direction: match[1], limit: Number(match[2] || 5), metric, label: match[3].trim() };
}

function teamGroups() {
    const ranking = parseRankingQuery(state.query);
    const query = ranking ? '' : state.query.trim().toLowerCase();
    const players = ranking ?
        state.players.slice().sort((a, b) => {
            const difference = Number(b[ranking.metric] || 0) - Number(a[ranking.metric] || 0);
            return ranking.direction === 'top' ? difference : -difference;
        }).slice(0, ranking.limit) :
        state.players;
    const groups = new Map(state.teams.map((team) => [team.id, { team, players: [] }]));
    players.forEach((player) => {
        const team = groups.get(player.contestantId) || { team: { id: player.contestantId, name: player.contestantName, shortName: player.contestantShortName, contestantFlagKey: player.contestantFlagKey }, players: [] };
        if (!groups.has(player.contestantId)) groups.set(player.contestantId, team);
        const matchesQuery = !query || [playerName(player), player.position, player.contestantName, player.contestantShortName].join(' ').toLowerCase().includes(query);
        if (matchesQuery) team.players.push(player);
    });
    return [...groups.values()].filter((group) => group.players.length).sort((a, b) => teamName(a.team).localeCompare(teamName(b.team)));
}

function render() {
    const groups = teamGroups();
    const ranking = parseRankingQuery(state.query);
    status.textContent = ranking ?
        `${ranking.direction} ${ranking.limit} players by ${ranking.label}` :
        `${state.players.length} players / ${groups.length} teams`;
    teamList.innerHTML = groups.length ? groups.map(renderTeam).join('') : '<div class="state-panel">No players match that search.</div>';
    bindTeamEvents();
    bindPlayerEvents();
}

function renderTeam(group) {
    const open = state.openTeam === group.team.id;
    const logoKey = group.team.contestantFlagKey || group.team.shirtKey || (group.players[0] && (group.players[0].contestantFlagKey || group.players[0].shirtKey));
    const logo = logoKey ?
        `<img class="team-logo" src="./images/${encodeURIComponent(logoKey)}.png" alt="${escapeHtml(teamName(group.team))} logo" onerror="this.hidden=true; this.nextElementSibling.hidden=false;" /><span class="team-logo-fallback" hidden>${escapeHtml(group.team.shortName || teamName(group.team).slice(0, 3).toUpperCase())}</span>` :
        `<span class="team-logo-fallback">${escapeHtml(group.team.shortName || teamName(group.team).slice(0, 3).toUpperCase())}</span>`;
    return `<article class="team-card ${open ? 'is-open' : ''}">
    <button class="team-heading" data-team="${escapeHtml(group.team.id)}" aria-expanded="${open}">
      <span class="team-title"><span class="team-badge">${logo}</span>${escapeHtml(teamName(group.team))}</span>
      <span class="team-count">${group.players.length} players <span class="team-arrow">+</span></span>
    </button>
    ${open ? renderTeamStats(group.team.id) : ''}
    ${open ? `<div class="player-list">${group.players.map(renderPlayer).join('')}</div>` : ''}
  </article>`;
}

function renderTeamStats(teamId) {
  const stats = state.teamStats[teamId];
  if (!stats) return '<div class="team-stats loading">Loading team metrics...</div>';
  if (stats.status === 'empty') return '';
  if (stats.status === 'error') return '<div class="team-stats empty">Team metrics are unavailable.</div>';
  return `<section class="team-stats">
    <h3>Team metrics</h3>
    <div class="detail-grid">
      <div class="metric"><span class="metric-label">Matches</span><span class="metric-value">${stats.matches}</span></div>
      <div class="metric"><span class="metric-label">Possession</span><span class="metric-value">${formatNumber(stats.possession)}%</span></div>
      <div class="metric"><span class="metric-label">Pass completion</span><span class="metric-value">${formatNumber(stats.passCompletionRate)}%</span></div>
      <div class="metric"><span class="metric-label">Shots</span><span class="metric-value">${stats.shots}</span></div>
      <div class="metric"><span class="metric-label">Shots on target</span><span class="metric-value">${stats.shotsOnTarget}</span></div>
      <div class="metric"><span class="metric-label">Corners</span><span class="metric-value">${stats.corners}</span></div>
      <div class="metric"><span class="metric-label">Fouls</span><span class="metric-value">${stats.fouls}</span></div>
      <div class="metric"><span class="metric-label">Tackles</span><span class="metric-value">${stats.tackles}</span></div>
      <div class="metric"><span class="metric-label">Saves</span><span class="metric-value">${stats.saves}</span></div>
      <div class="metric"><span class="metric-label">Yellow cards</span><span class="metric-value">${stats.yellowCards}</span></div>
      <div class="metric"><span class="metric-label">Red cards</span><span class="metric-value">${stats.redCards}</span></div>
    </div>
  </section>`;
}

function renderPlayer(player) {
  const open = state.openPlayer === player.playerId;
  const availability = String(player.availabilityDisplay || '').toLowerCase();
  const unavailable = availability.includes('injured') || availability.includes('suspended');
  return `<div class="player-row ${open ? 'is-open' : ''} ${unavailable ? 'is-unavailable' : ''}">
    <button class="player-toggle" data-player="${escapeHtml(player.playerId)}" aria-expanded="${open}">
      <span><span class="player-name">${escapeHtml(playerName(player))} <span class="player-price">£${formatNumber(player.price)}m</span></span><span class="player-position">${escapeHtml(player.position)} / ${escapeHtml(player.availabilityDisplay)}</span></span>
      <span class="player-points">${formatNumber(player.totalPoints)} pts</span>
    </button>
    ${open ? `<div class="player-detail" id="detail-${escapeHtml(player.playerId)}">${renderPlayerSummary(player)}</div>` : ''}
  </div>`;
}

function renderPlayerSummary(player) {
  const isGoalkeeper = player.position === 'GK';
  const isDefender = player.position === 'DEF';
  const metric = (label, value, suffix = '', prefix = '') => `<div class="metric"><span class="metric-label">${label}</span><span class="metric-value">${prefix}${formatNumber(value)}${suffix}</span></div>`;
  const group = (title, metrics) => `<section class="metric-group"><h3 class="metric-group-title">${title}</h3><div class="detail-grid">${metrics.join('')}</div></section>`;
  const summaryMetrics = [
    metric('Price', player.price, 'm', '£'),
    metric('Average', player.averagePoints),
    metric('Last 3 average', player.last3Average),
    metric('PPM points', player.ppmPoints),
    metric('Bonus points', player.bonusPoints),
    metric('Total points', player.totalPoints),
  ];
  const scoringMetrics = [
    metric('Goals', player.goals),
    metric('Assists', player.assists),
    metric('Shots on target', player.shotsOnTarget),
    metric('Big chances created', player.chancesCreated),
    metric('Tackles', player.tackles),
    metric('Yellow cards', player.yellowCards),
    metric('Red cards', player.redCards),
    metric('Penalty misses', player.penaltyMisses),
    metric('Own goals', player.ownGoals),
  ];
  if (isGoalkeeper || isDefender) {
    scoringMetrics.push(metric('Goals conceded', player.goalsConceded), metric('Clean sheet', player.cleanSheet));
  }
  if (isGoalkeeper) scoringMetrics.push(metric('Saves', player.saves), metric('Penalty saves', player.penaltySaves));
  const ppmMetrics = [
    metric('Dribbles', player.dribbles),
    metric('Crosses', player.crosses),
    metric('Interceptions', player.interceptions),
    metric('Blocks', player.blocks),
    metric('Fouls won', player.foulsWon),
    metric('Pass completion rate', player.passCompletionRate, '%'),
    metric('Offsides', player.offsides),
    metric('Fouls conceded', player.foulsMade),
    metric('Errors', player.errorsLeadingToGoal),
  ];
  if (isGoalkeeper) ppmMetrics.push(metric('Claims', player.claims), metric('Punches', player.punches), metric('Keeper sweeps', player.keeperSweeps));
  return `${group('Player Summary', summaryMetrics)}${group('Scoring Metrics', scoringMetrics)}${group('PPM Metrics', ppmMetrics)}
  <div class="detail-section"><h3>Next fixtures</h3><div class="fixture-list">${renderFixtures(player.nextGameweekFixtures)}</div></div>
  <div class="detail-section match-history"><h3>Match history</h3><div class="loading">Loading match history...</div></div>`;
}

async function loadTeamStats(group) {
  const teamId = group.team.id;
  if (state.teamStats[teamId]) return;
  state.teamStats[teamId] = { status: 'loading' };
  try {
    const player = group.players[0];
    const matchesResponse = await fetch(`${dataUri}${player.playerId}-matches.json`, { cache: 'no-store' });
    if (!matchesResponse.ok) throw new Error('Team match history unavailable');
    const matchesPayload = await matchesResponse.json();
    const matchIds = [...new Set((matchesPayload.data?.items || []).map((match) => match.matchId).filter(Boolean))];
    const statsPayloads = await Promise.all(matchIds.map(async (matchId) => {
      const response = await fetch(`${dataUri}${matchId}-result-stats.json`, { cache: 'no-store' });
      if (!response.ok) return null;
      return response.json();
    }));
    const teamMatches = statsPayloads
      .map((payload) => [payload?.data?.home, payload?.data?.away].find((stats) => stats?.contestantId === teamId))
      .filter(Boolean);
    if (!teamMatches.length) {
      state.teamStats[teamId] = { status: 'empty' };
    } else {
      const total = (field) => teamMatches.reduce((sum, match) => sum + Number(match[field] || 0), 0);
      state.teamStats[teamId] = {
        status: 'ready',
        matches: teamMatches.length,
        possession: total('possession') / teamMatches.length,
        passCompletionRate: total('passCompletionRate') / teamMatches.length,
        shots: total('shots'),
        shotsOnTarget: total('shotsOnTarget'),
        corners: total('corners'),
        fouls: total('fouls'),
        tackles: total('tackles'),
        saves: total('saves'),
        yellowCards: total('yellowCards'),
        redCards: total('redCards'),
      };
    }
  } catch (error) {
    state.teamStats[teamId] = { status: 'error' };
  }
  if (state.openTeam === teamId) render();
}

function renderFixtures(fixtures = []) {
  if (!fixtures.length) return '<span class="empty">No upcoming fixtures.</span>';
  return fixtures.map((fixture) => `<div class="fixture"><span>${fixture.isHome ? 'vs' : '@'} ${escapeHtml(fixture.opponentShortName || fixture.opponentName)}</span><span class="match-meta">GW ${fixture.gameweek} / ${new Date(fixture.kickoffAt).toLocaleDateString('en-GB', { day: '2-digit', month: 'short' })}</span></div>`).join('');
}

function renderMatch(match) {
  const stats = match.stats?.length
    ? match.stats.map((stat) => `<div class="match-stat"><span>${escapeHtml(stat.label)}</span><span>Total: ${escapeHtml(stat.total)} <strong>${escapeHtml(stat.points)} pts</strong></span></div>`).join('')
    : '<span class="empty">No player stats recorded.</span>';
  const matchPoints = match.mdPoints || '0';
  return `<details class="match">
    <summary><span>${escapeHtml(match.leftTeam?.shortName)} ${match.leftTeam?.score ?? '-'} - ${match.rightTeam?.score ?? '-'} ${escapeHtml(match.rightTeam?.shortName)} <strong class="match-points">(${escapeHtml(matchPoints)} pts)</strong></span><span class="match-meta">${escapeHtml(match.competitionLabel)} / ${new Date(match.kickoffAt).toLocaleDateString('en-GB', { day: '2-digit', month: 'short' })} <span class="match-toggle">View stats</span></span></summary>
    <div class="match-stats">${stats}</div>
  </details>`;
}

async function loadMatches(player) {
  const detail = document.querySelector(`#detail-${CSS.escape(player.playerId)}`);
  if (!detail) return;
  try {
    const response = await fetch(`${dataUri}${player.playerId}-matches.json`, { cache: 'no-store' });
    if (!response.ok) throw new Error('Match history unavailable');
    const payload = await response.json();
    const history = detail.querySelector('.match-history');
    history.innerHTML = `<h3>Match history</h3>${payload.data?.items?.length ? payload.data.items.slice(0, 5).map(renderMatch).join('') : '<span class="empty">No match history.</span>'}`;
  } catch (error) {
    detail.querySelector('.match-history').innerHTML = '<h3>Match history</h3><span class="empty">Match history is unavailable.</span>';
  }
}

function bindTeamEvents() {
  document.querySelectorAll('[data-team]').forEach((button) => button.addEventListener('click', () => {
    state.openTeam = state.openTeam === button.dataset.team ? null : button.dataset.team;
    state.openPlayer = null;
    render();
    if (state.openTeam) {
      const group = teamGroups().find((candidate) => candidate.team.id === state.openTeam);
      if (group) loadTeamStats(group);
    }
  }));
}

function bindPlayerEvents() {
  document.querySelectorAll('[data-player]').forEach((button) => button.addEventListener('click', () => {
    const id = button.dataset.player;
    state.openPlayer = state.openPlayer === id ? null : id;
    render();
    if (state.openPlayer) loadMatches(state.players.find((player) => player.playerId === id));
  }));
}

async function init() {
  try {
    const [playersResponse, teamsResponse] = await Promise.all([
      fetch(`${dataUri}players.json`, { cache: 'no-store' }),
      fetch(`${dataUri}teams.json`, { cache: 'no-store' }),
    ]);
    if (!playersResponse.ok || !teamsResponse.ok) throw new Error('Could not load data files');
    state.players = await playersResponse.json();
    const teamsPayload = await teamsResponse.json();
    state.teams = teamsPayload.data || [];
    render();
  } catch (error) {
    errorState.textContent = 'The data directory could not be loaded. Start the site through a local web server so the JSON files can be fetched.';
    errorState.classList.remove('is-hidden');
    status.textContent = 'Data unavailable';
  }
}

search.addEventListener('input', (event) => { state.query = event.target.value; state.openTeam = null; state.openPlayer = null; render(); });
init();