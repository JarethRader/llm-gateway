import * as React from "react"
import { Badge } from "@/components/ui/badge"

import { cn } from "@/lib/utils"
import { cva, VariantProps } from "class-variance-authority"

const statusPillVariants = cva(
  "flex flex-wrap gap-2 content-center",
  {
    variants: {
      status: {
        closed: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
        "half-open": "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
        open: "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300",
        disabled: ""
      }
    },
    defaultVariants: {
      status: "open",
    }
  }
)

function StatusPill({
  className,
  status,
  ...props
}: React.ComponentProps<typeof Badge> & VariantProps<typeof statusPillVariants>) {

  return (
    <Badge
      data-slot="status-pill"
      data-status={status}
      className={cn(statusPillVariants({ status }), className)}
      variant="secondary"
      {...props}
    />
  )
}

export { StatusPill, statusPillVariants }
