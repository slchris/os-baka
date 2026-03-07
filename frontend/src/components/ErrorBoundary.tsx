import React from 'react';
import { AlertTriangle, RefreshCw, Home } from 'lucide-react';

interface ErrorBoundaryProps {
    children: React.ReactNode;
    fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
    errorInfo: React.ErrorInfo | null;
}

/**
 * Global error boundary that catches React rendering errors
 * and shows a recovery UI instead of a blank white screen.
 */
class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null, errorInfo: null };
    }

    static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
        this.setState({ errorInfo });
        // Log to console in development
        console.error('[ErrorBoundary] Caught error:', error, errorInfo);
    }

    handleReset = () => {
        this.setState({ hasError: false, error: null, errorInfo: null });
    };

    handleGoHome = () => {
        window.location.href = '/';
    };

    render() {
        if (this.state.hasError) {
            if (this.props.fallback) {
                return this.props.fallback;
            }

            return (
                <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    minHeight: '100vh',
                    background: 'var(--bg-primary, #0a0a0a)',
                    color: 'var(--text-primary, #e5e5e5)',
                    fontFamily: '-apple-system, BlinkMacSystemFont, "Inter", sans-serif',
                    padding: '2rem',
                }}>
                    <div style={{
                        maxWidth: '480px',
                        textAlign: 'center',
                    }}>
                        <div style={{
                            width: '64px',
                            height: '64px',
                            borderRadius: '16px',
                            background: 'rgba(239, 68, 68, 0.1)',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            margin: '0 auto 1.5rem',
                        }}>
                            <AlertTriangle size={32} color="#ef4444" />
                        </div>

                        <h1 style={{
                            fontSize: '1.5rem',
                            fontWeight: 600,
                            marginBottom: '0.5rem',
                        }}>
                            Something went wrong
                        </h1>

                        <p style={{
                            color: 'var(--text-secondary, #a3a3a3)',
                            fontSize: '0.9rem',
                            lineHeight: 1.6,
                            marginBottom: '1.5rem',
                        }}>
                            An unexpected error occurred. You can try refreshing the page or
                            navigating back to the dashboard.
                        </p>

                        {this.state.error && (
                            <pre style={{
                                background: 'rgba(255, 255, 255, 0.05)',
                                borderRadius: '8px',
                                padding: '1rem',
                                fontSize: '0.75rem',
                                textAlign: 'left',
                                overflow: 'auto',
                                maxHeight: '120px',
                                marginBottom: '1.5rem',
                                border: '1px solid rgba(255, 255, 255, 0.1)',
                                color: '#ef4444',
                            }}>
                                {this.state.error.message}
                            </pre>
                        )}

                        <div style={{ display: 'flex', gap: '0.75rem', justifyContent: 'center' }}>
                            <button
                                onClick={this.handleReset}
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '0.5rem',
                                    padding: '0.6rem 1.2rem',
                                    borderRadius: '8px',
                                    border: '1px solid rgba(255, 255, 255, 0.15)',
                                    background: 'rgba(255, 255, 255, 0.08)',
                                    color: 'inherit',
                                    cursor: 'pointer',
                                    fontSize: '0.85rem',
                                    fontWeight: 500,
                                }}
                            >
                                <RefreshCw size={16} />
                                Try Again
                            </button>
                            <button
                                onClick={this.handleGoHome}
                                style={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: '0.5rem',
                                    padding: '0.6rem 1.2rem',
                                    borderRadius: '8px',
                                    border: 'none',
                                    background: '#3b82f6',
                                    color: '#fff',
                                    cursor: 'pointer',
                                    fontSize: '0.85rem',
                                    fontWeight: 500,
                                }}
                            >
                                <Home size={16} />
                                Go to Dashboard
                            </button>
                        </div>
                    </div>
                </div>
            );
        }

        return this.props.children;
    }
}

export default ErrorBoundary;
