import { Component, type ReactNode } from "react";
import { href } from "../lib/route";

interface Props {
  /** Clears the error when it changes (the URL hash, so any navigation retries). */
  resetKey: string;
  children: ReactNode;
}

/**
 * Spec content is untrusted and can be malformed in ways the views don't
 * expect. Without a boundary one bad value unmounts the whole app until a
 * reload; with it, only the current page shows an error.
 */
export class ErrorBoundary extends Component<Props, { error: Error | null; key: string }> {
  state = { error: null as Error | null, key: this.props.resetKey };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  static getDerivedStateFromProps(props: Props, state: { error: Error | null; key: string }) {
    return props.resetKey === state.key ? null : { error: null, key: props.resetKey };
  }

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;
    return (
      <div className="empty" role="alert">
        <p>This page couldn't be displayed. The spec may contain something the UI doesn't handle.</p>
        <pre className="code">{error.message}</pre>
        <p>
          <a href={href.home()}>Back to all specs</a>
        </p>
      </div>
    );
  }
}
