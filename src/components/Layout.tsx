import React from "react";
import { Container } from "reactstrap";
import BgWrapper from "./BgWrapper";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <BgWrapper>
      <Container fluid style={{ padding: 0 }}>
        <div>{children}</div>
      </Container>
    </BgWrapper>
  );
}

Layout.displayName = "Layout";
