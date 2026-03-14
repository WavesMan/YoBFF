import * as React from "react"
import "./Switch.css"

export interface SwitchProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  checked?: boolean
  onCheckedChange?: (checked: boolean) => void
}

export const Switch = React.forwardRef<HTMLButtonElement, SwitchProps>(
  ({ className, checked, onCheckedChange, disabled, ...props }, ref) => {
    return (
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        data-state={checked ? "checked" : "unchecked"}
        disabled={disabled}
        ref={ref}
        onClick={() => onCheckedChange?.(!checked)}
        className={`switch-root ${className || ""}`}
        {...props}
      >
        <span
          data-state={checked ? "checked" : "unchecked"}
          className="switch-thumb"
        />
      </button>
    )
  }
)
Switch.displayName = "Switch"
