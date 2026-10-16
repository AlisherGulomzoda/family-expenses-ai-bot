import { useRef, useState } from 'react'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

/**
 * One line of text cut with an ellipsis. The full text appears in a tooltip,
 * but only when the text is actually cut (design-spec §4 «Комментарий»).
 * The trigger is a plain-looking button, so keyboard users can focus it and
 * read the tooltip too.
 */
export function TruncatedText({ text }: { text: string }) {
  const ref = useRef<HTMLButtonElement>(null)
  const [open, setOpen] = useState(false)
  const isCut = () => {
    const el = ref.current
    return !!el && el.scrollWidth > el.clientWidth
  }
  return (
    <Tooltip open={open} onOpenChange={(next) => setOpen(next && isCut())}>
      <TooltipTrigger asChild>
        <button
          ref={ref}
          type="button"
          className="block w-full cursor-default truncate rounded-sm text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          {text}
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-[360px] break-words">{text}</TooltipContent>
    </Tooltip>
  )
}
