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
  const lesson = vi.fn()
  component.$on('depth', depth)
  component.$on('resume', resume)
  component.$on('review', review)
  component.$on('explore', explore)
  component.$on('lesson', lesson)

  expect(screen.getByRole('heading', { name: 'Italian Game for White' })).toBeInTheDocument()
  expect(screen.getByLabelText('Course depth')).toHaveValue('reference')
  expect(screen.getByText('1 of 3 lessons complete')).toBeInTheDocument()
  await fireEvent.change(screen.getByLabelText('Course depth'), { target: { value: 'quick' } })
  expect(depth).toHaveBeenCalledWith(expect.objectContaining({
    detail: { courseId: 'synthetic-italian', depth: 'quick' }
  }))

  await fireEvent.click(screen.getByRole('button', { name: 'Continue learning — Giuoco Piano' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Review 3 due positions' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Explore variations' }))
  expect(resume).toHaveBeenCalledOnce()
  expect(review).toHaveBeenCalledWith(expect.objectContaining({ detail: 'synthetic-italian' }))
  expect(explore).toHaveBeenCalledWith(expect.objectContaining({
    detail: { courseId: 'synthetic-italian', positionId: 'initial' }
  }))
  await fireEvent.click(screen.getByRole('button', { name: 'Study Giuoco Piano' }))
  expect(lesson).toHaveBeenCalledWith(expect.objectContaining({
    detail: { courseId: 'synthetic-italian', lessonId: 'giuoco-c3' }
  }))
})

test('starts the recommended lesson when there is no resumable activity', async () => {
  const { component } = render(OpeningHub, { home: fakeOpeningHome })
  const lesson = vi.fn()
  component.$on('lesson', lesson)

  await fireEvent.click(screen.getByRole('button', { name: 'Continue learning — Giuoco Piano' }))
  expect(lesson).toHaveBeenCalledWith(expect.objectContaining({
    detail: { courseId: 'synthetic-italian', lessonId: 'giuoco-c3' }
  }))
})

test('keeps a paused review separate from Continue learning', async () => {
  const home = {
    ...fakeOpeningHome,
    courses: [{
      ...fakeOpeningHome.courses[0],
      hasResumable: false,
      hasResumableReview: true,
      dueReviews: 1
    }]
  }
  const { component } = render(OpeningHub, { home })
  const lesson = vi.fn()
  const resume = vi.fn()
  const review = vi.fn()
  component.$on('lesson', lesson)
  component.$on('resume', resume)
  component.$on('review', review)

  await fireEvent.click(screen.getByRole('button', { name: 'Continue learning — Giuoco Piano' }))
  expect(lesson).toHaveBeenCalledOnce()
  expect(resume).not.toHaveBeenCalled()

  await fireEvent.click(screen.getByRole('button', { name: 'Continue review — 1 due position' }))
  expect(review).toHaveBeenCalledWith(expect.objectContaining({ detail: 'synthetic-italian' }))
})

test('keeps a paused review available when the selected depth has no due positions', async () => {
  const home = {
    ...fakeOpeningHome,
    courses: [{
      ...fakeOpeningHome.courses[0],
      hasResumableReview: true,
      dueReviews: 0
    }]
  }
  const { component } = render(OpeningHub, { home })
  const review = vi.fn()
  component.$on('review', review)

  await fireEvent.click(screen.getByRole('button', { name: 'Continue review' }))

  expect(review).toHaveBeenCalledWith(expect.objectContaining({ detail: 'synthetic-italian' }))
})

test('shows private import guidance when no course is installed', () => {
  render(OpeningHub, { home: { notice: 'Course storage is available.', courses: [] } })
  expect(screen.getByText('Import a private .ctcourse file from Parent settings.')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /Start / })).not.toBeInTheDocument()
  expect(screen.getByText('Course storage is available.')).toBeInTheDocument()
})

test('renders the explicitly selected course when several courses are available', async () => {
	const second = {
		...fakeOpeningHome.courses[0],
		courseId: 'queens-gambit-white',
		title: "Queen's Gambit for White",
		recommendedLessonTitle: 'Build the d4 centre'
	}
	const { component } = render(OpeningHub, {
		home: { courses: [fakeOpeningHome.courses[0], second] },
		selectedCourseId: second.courseId
	})
	const select = vi.fn()
	component.$on('select', select)

	expect(screen.getByRole('heading', { name: "Queen's Gambit for White" })).toBeInTheDocument()
	expect(screen.queryByRole('heading', { name: 'Italian Game for White' })).not.toBeInTheDocument()
	await fireEvent.change(screen.getByLabelText('Opening course'), {
		target: { value: 'synthetic-italian' }
	})
	expect(select).toHaveBeenCalledWith(expect.objectContaining({ detail: 'synthetic-italian' }))
})
