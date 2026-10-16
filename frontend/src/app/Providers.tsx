import { QueryClientProvider, type QueryClient } from '@tanstack/react-query'
import { ThemeProvider } from 'next-themes'
import { useState, type ReactNode } from 'react'

import { createQueryClient } from '@/app/queryClient'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useIsDesktop } from '@/shared/useMediaQuery'

/** App-wide providers: theme, server state, tooltips and toasts. */
export function Providers({ children, queryClient }: { children: ReactNode; queryClient?: QueryClient }) {
  const [client] = useState(() => queryClient ?? createQueryClient())
  return (
    // The design switches themes with the .dark class on <html>.
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <QueryClientProvider client={client}>
        <TooltipProvider>
          {children}
          <AppToaster />
        </TooltipProvider>
      </QueryClientProvider>
    </ThemeProvider>
  )
}

/** Toasts: bottom right on desktop, top center on phones so the floating «+» stays visible. */
function AppToaster() {
  const isDesktop = useIsDesktop()
  return <Toaster position={isDesktop ? 'bottom-right' : 'top-center'} />
}
