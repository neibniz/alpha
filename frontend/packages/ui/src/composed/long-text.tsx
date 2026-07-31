import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@workspace/ui/components/popover";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@workspace/ui/components/tooltip";
import { cn } from "@workspace/ui/lib/utils";
import { useEffect, useRef, useState } from "react";

type LongTextProps = {
  children: React.ReactNode;
  className?: string;
  contentClassName?: string;
};

export function LongText({
  children,
  className = "",
  contentClassName = "",
}: LongTextProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [isOverflown, setIsOverflown] = useState(false);

  useEffect(() => {
    if (checkOverflow(ref.current)) {
      setIsOverflown(true);
      return;
    }

    setIsOverflown(false);
  }, []);

  if (!isOverflown)
    return (
      <div className={cn("truncate", className)} ref={ref}>
        {children}
      </div>
    );

  return (
    <>
      <div className="hidden sm:block">
        <TooltipProvider delayDuration={0}>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className={cn("truncate", className)} ref={ref}>
                {children}
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p className={cn("whitespace-pre-line", contentClassName)}>
                {children}
              </p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
      <div className="sm:hidden">
        <Popover>
          <PopoverTrigger asChild>
            <div className={cn("truncate", className)} ref={ref}>
              {children}
            </div>
          </PopoverTrigger>
          <PopoverContent
            className={cn("w-fit whitespace-pre-line", contentClassName)}
          >
            <p>{children}</p>
          </PopoverContent>
        </Popover>
      </div>
    </>
  );
}

const checkOverflow = (textContainer: HTMLDivElement | null) => {
  if (textContainer) {
    return (
      textContainer.offsetHeight < textContainer.scrollHeight ||
      textContainer.offsetWidth < textContainer.scrollWidth
    );
  }
  return false;
};
