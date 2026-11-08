import React from "react";

export default function AnimatedBackground() {
  return (
    <div className="ambient" aria-hidden="true">
      <div className="ambient-blob blob-one" />
      <div className="ambient-blob blob-two" />
      <div className="ambient-blob blob-three" />
      <div className="ambient-grain" />
    </div>
  );
}
