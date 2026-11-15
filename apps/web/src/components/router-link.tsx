import { Link } from "@tanstack/react-router";
import type { AnchorHTMLAttributes } from "react";

// Astryx renders every link through this component, so links use client-side routing.
export const RouterLink = ({
  href,
  ...props
}: AnchorHTMLAttributes<HTMLAnchorElement>) => <Link to={href} {...props} />;
