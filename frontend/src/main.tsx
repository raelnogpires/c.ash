import { Component, StrictMode, type ErrorInfo, type ReactNode } from 'react'
import { createRoot } from 'react-dom/client'
import '@fontsource-variable/inter'
import App from './App'
import './styles.css'

class StartupErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Unable to start [c]ash', error, info)
  }

  render() {
    if (this.state.error) {
      return <main className="fatal"><div className="brand"><span>[c]</span>ash</div><h1>Não conseguimos iniciar o aplicativo</h1><p role="alert">{this.state.error.message || 'Ocorreu um erro inesperado.'}</p></main>
    }
    return this.props.children
  }
}

createRoot(document.getElementById('root')!).render(<StrictMode><StartupErrorBoundary><App /></StartupErrorBoundary></StrictMode>)
