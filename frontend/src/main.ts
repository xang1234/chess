import '@lichess-org/chessground/assets/chessground.base.css'
import '@lichess-org/chessground/assets/chessground.cburnett.css'
import './styles/tokens.css'
import './styles/app.css'
import './styles/chessground-theme.css'
import App from './App.svelte'

const app = new App({
  target: document.getElementById('app')
})

export default app
