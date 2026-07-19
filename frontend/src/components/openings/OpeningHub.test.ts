import { fireEvent, render, screen } from '@testing-library/svelte'
import { vi } from 'vitest'
import OpeningHub from './OpeningHub.svelte'
import { fakeOpeningHome } from '../../test-fakes'

test('shows depth, progress, ordered lessons, review, and explorer actions', async () => {
  const home = {
    ...fakeOpeningHome,
    courses: [{
      ...fakeOpeningHome.courses[0],
      hasResumable: true,
      nextLessonTitle: 'Giuoco Piano',
      dueReviews: 3
    }]
  }
  const { component } = render(OpeningHub, { home })
  const depth = vi.fn()
  const resume = vi.fn()
  const review = vi.fn()
  const explore = vi.fn()
  component.$on('depth', depth)
  component.$on('resume', resume)
  component.$on('review', review)
  component.$on('explore', explore)

  expect(screen.getByRole('heading', { name: 'Italian Game for White' })).toBeInTheDocument()
  expect(screen.getByLabelText('Course depth')).toHaveValue('reference')
  expect(screen.getByText('1 of 3 lessons complete')).toBeInTheDocument()
  await fireEvent.change(screen.getByLabelText('Course depth'), { target: { value: 'quick' } })
  expect(depth).toHaveBeenCalledWith(expect.objectContaining({
    detail: { courseId: 'synthetic-italian', depth: 'quick' }
  }))

  await fireEvent.click(screen.getByRole('button', { name: 'Continue Giuoco Piano' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Review 3 due positions' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Explore variations' }))
  expect(resume).toHaveBeenCalledOnce()
  expect(review).toHaveBeenCalledWith(expect.objectContaining({ detail: 'synthetic-italian' }))
  expect(explore).toHaveBeenCalledWith(expect.objectContaining({
    detail: { courseId: 'synthetic-italian', positionId: 'initial' }
  }))
  expect(screen.getByRole('button', { name: 'Start Giuoco Piano' })).toBeInTheDocument()
})

test('shows private import guidance when no course is installed', () => {
  render(OpeningHub, { home: { notice: 'Course storage is available.', courses: [] } })
  expect(screen.getByText('Import a private .ctcourse file from Parent settings.')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /Start / })).not.toBeInTheDocument()
  expect(screen.getByText('Course storage is available.')).toBeInTheDocument()
})
