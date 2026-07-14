import { render, screen } from '@testing-library/svelte'
import App from './App.svelte'

test('renders the product name', () => {
  render(App)
  expect(screen.getByText('Chess Trainer')).toBeInTheDocument()
})
