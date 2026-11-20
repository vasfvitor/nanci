// warningBannerClass colors a q-banner that warns, in the light or dark theme.
export function warningBannerClass(dark: boolean) {
  return dark ? 'bg-grey-9 text-orange-3' : 'bg-orange-1 text-orange-10'
}

// mutedBannerClass colors a q-banner that only informs, in the light or dark
// theme.
export function mutedBannerClass(dark: boolean) {
  return dark ? 'bg-grey-9 text-grey-4' : 'bg-grey-2 text-grey-8'
}
