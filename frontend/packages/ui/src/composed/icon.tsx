"use client";

// import { icons as FlagPack } from '@iconify-json/flagpack';
// import { icons as Logos } from '@iconify-json/logos';
// import { icons as Mdi } from '@iconify-json/mdi';
// import { icons as Simple } from '@iconify-json/simple-icons';
// import { icons as Uil } from '@iconify-json/uil';

import { Icon as Iconify, type IconProps } from "@iconify/react";

// addCollection(FlagPack);
// addCollection(Mdi);
// addCollection(Uil);
// addCollection(Simple);
// addCollection(Logos);

export function Icon(props: IconProps) {
  return <Iconify {...props} />;
}
