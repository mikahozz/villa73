import React from "react";
import BgWrapper from "./BgWrapper";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <BgWrapper>
      <main className="w-full p-0">{children}</main>
    </BgWrapper>
  );
}

Layout.displayName = "Layout";
