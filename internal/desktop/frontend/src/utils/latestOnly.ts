// LatestGate tells a load whether a newer one started after it.
export type LatestGate = {
  // begin starts a load and returns its ticket: a check that stays true
  // until the next begin.
  begin(): () => boolean
}

// latestOnly returns a gate for loads that may end out of order, such as
// searches started by quick filter changes; only the latest one may write
// its result.
export function latestOnly(): LatestGate {
  let latest = 0
  return {
    begin() {
      const ticket = ++latest
      return () => ticket === latest
    },
  }
}
