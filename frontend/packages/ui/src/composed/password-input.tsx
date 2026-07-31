import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@workspace/ui/components/input-group";
import { Eye, EyeOff } from "lucide-react";
import * as React from "react";

type PasswordInputProps = Omit<
  React.InputHTMLAttributes<HTMLInputElement>,
  "type"
> & {
  ref?: React.Ref<HTMLInputElement>;
};

export function PasswordInput({
  className,
  disabled,
  ref,
  ...props
}: PasswordInputProps) {
  const [showPassword, setShowPassword] = React.useState(false);

  return (
    <InputGroup className={className}>
      <InputGroupInput
        disabled={disabled}
        ref={ref}
        type={showPassword ? "text" : "password"}
        {...props}
      />
      <InputGroupAddon align="inline-end">
        <InputGroupButton
          aria-label={showPassword ? "Hide password" : "Show password"}
          disabled={disabled}
          onClick={() => setShowPassword((prev) => !prev)}
          size="icon-xs"
        >
          {showPassword ? <EyeOff /> : <Eye />}
        </InputGroupButton>
      </InputGroupAddon>
    </InputGroup>
  );
}
