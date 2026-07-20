import { render, screen } from '@testing-library/svelte'
import OpeningPathContext from './OpeningPathContext.svelte'

test('renders the learner path in order as course context', () => {
  render(OpeningPathContext, {
    path: [
      { lessonId: 'foundation', title: 'Reach the Italian' },
      { lessonId: 'split', title: 'Choose Black’s setup' },
      { lessonId: 'giuoco', title: 'Prepare d4 with c3' }
    ]
  })

  const path = screen.getByRole('navigation', { name: 'Opening course path' })
  expect(path).toHaveTextContent('Reach the Italian')
  expect(path.textContent?.indexOf('Choose Black’s setup'))
    .toBeLessThan(path.textContent?.indexOf('Prepare d4 with c3') ?? 0)
})
