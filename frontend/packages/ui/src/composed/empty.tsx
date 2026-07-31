import {
  Empty as EmptyContainer,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
} from "@workspace/ui/components/empty";
import { cn } from "@workspace/ui/lib/utils";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

export default function Empty({
  description,
  border,
}: {
  description?: React.ReactNode;
  border?: boolean;
}) {
  const { t } = useTranslation("components");

  const messages = useMemo(
    () => [
      t(
        "empty.tips.0",
        "Imagine this space filled with exciting content! For now, you'll have to use your imagination..."
      ),
      t(
        "empty.tips.1",
        "This area mysteriously disappeared, but we're summoning it back!"
      ),
      t(
        "empty.tips.2",
        "Oh no, nothing happened... Feel free to fill in the blank!"
      ),
      t(
        "empty.tips.3",
        "It's like discovering an empty stage at a concert... Why not go up and perform?"
      ),
      t(
        "empty.tips.4",
        "You've found a blank canvas! How about building a house?"
      ),
      t(
        "empty.tips.5",
        "This area is currently empty, but creativity starts here!"
      ),
      t(
        "empty.tips.6",
        "Nothing here... but don't worry, it's just the beginning!"
      ),
      t(
        "empty.tips.7",
        "This place was supposed to have a big surprise, but the surprise slipped away!"
      ),
      t(
        "empty.tips.8",
        "There's nothing here for now, like an empty snack cabinet."
      ),
      t(
        "empty.tips.9",
        "This empty space is waiting for its protagonist to take the stage!"
      ),
    ],
    [t]
  );

  const [index, setIndex] = useState(0);

  useEffect(() => {
    setIndex(Math.floor(Math.random() * messages.length));
  }, [messages]);

  return (
    <EmptyContainer className={cn(border ? "border" : "border-none")}>
      <EmptyMedia>
        <svg
          className="text-background"
          fill="currentColor"
          height="41"
          stroke="currentColor"
          viewBox="0 0 64 41"
          width="64"
          xmlns="http://www.w3.org/2000/svg"
        >
          <title>{t("empty.title", "Empty")}</title>
          <g fill="none" fillRule="evenodd" transform="translate(0 1)">
            <ellipse
              cx="32"
              cy="33"
              fill="currentColor"
              opacity={0.8}
              rx="32"
              ry="7"
            />
            <g fillRule="nonzero" stroke="#d9d9d9">
              <path d="M55 12.76L44.854 1.258C44.367.474 43.656 0 42.907 0H21.093c-.749 0-1.46.474-1.947 1.257L9 12.761V22h46v-9.24z" />
              <path
                d="M41.613 15.931c0-1.605.994-2.93 2.227-2.931H55v18.137C55 33.26 53.68 35 52.05 35h-40.1C10.32 35 9 33.259 9 31.137V13h11.16c1.233 0 2.227 1.323 2.227 2.928v.022c0 1.605 1.005 2.901 2.237 2.901h14.752c1.232 0 2.237-1.308 2.237-2.913v-.007z"
                fill="currentColor"
              />
            </g>
          </g>
        </svg>
      </EmptyMedia>
      <EmptyHeader>
        <EmptyDescription>{description || messages[index]}</EmptyDescription>
      </EmptyHeader>
    </EmptyContainer>
  );
}
