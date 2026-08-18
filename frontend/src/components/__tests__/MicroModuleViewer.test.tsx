import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MicroModuleViewer, { MicroModuleData } from '../MicroModuleViewer';

const quizModules: MicroModuleData[] = [
  {
    id: 'mm-1',
    title: 'Truth Values',
    content_text: 'Two truth values exist.',
    question: 'How many truth values exist in classical logic?',
    options: ['One', 'Two', 'Three'],
    correct_index: 1,
    explanation: 'Classical logic works with True and False.',
  },
  {
    id: 'mm-2',
    title: 'Negation',
    content_text: 'NOT flips a value.',
    question: 'What is NOT(False)?',
    options: ['True', 'False'],
    correct_index: 0,
    explanation: 'Negation always flips.',
  },
];

describe('MicroModuleViewer', () => {
  it('renders the first module content with a knowledge check', () => {
    render(<MicroModuleViewer modules={quizModules} onComplete={jest.fn()} />);
    expect(screen.getByText('Truth Values')).toBeInTheDocument();
    expect(screen.getByText('Knowledge Check')).toBeInTheDocument();
    expect(screen.getByText('How many truth values exist in classical logic?')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'One' })).toBeInTheDocument();
  });

  it('blocks Next until the quiz is answered correctly', async () => {
    const user = userEvent.setup();
    render(<MicroModuleViewer modules={quizModules} onComplete={jest.fn()} />);

    const nextButton = screen.getByRole('button', { name: /Next/ });
    expect(nextButton).toBeDisabled();

    // Wrong answer: stays locked, supportive feedback shows, retry allowed.
    await user.click(screen.getByRole('button', { name: 'One' }));
    expect(screen.getByText('Not quite — give it another try!')).toBeInTheDocument();
    expect(nextButton).toBeDisabled();

    // Correct answer unlocks Next and shows the explanation.
    await user.click(screen.getByRole('button', { name: 'Two' }));
    expect(screen.getByText('Correct!')).toBeInTheDocument();
    expect(screen.getByText('Classical logic works with True and False.')).toBeInTheDocument();
    expect(nextButton).toBeEnabled();
  });

  it('reports honest attempt facts: elapsed time, first-try correct, total checks', async () => {
    const user = userEvent.setup();
    const onComplete = jest.fn();

    render(<MicroModuleViewer modules={quizModules} onComplete={onComplete} />);

    // Module 1: one wrong first try, then correct.
    await user.click(screen.getByRole('button', { name: 'One' }));
    await user.click(screen.getByRole('button', { name: 'Two' }));
    await user.click(screen.getByRole('button', { name: /Next/ }));

    // Module 2: correct first try.
    await user.click(screen.getByRole('button', { name: 'True' }));
    await user.click(screen.getByRole('button', { name: /Complete/ }));

    expect(onComplete).toHaveBeenCalledTimes(1);
    const stats = onComplete.mock.calls[0][0];
    expect(stats.total_count).toBe(2);
    expect(stats.correct_count).toBe(1); // only the first try counts
    expect(stats.elapsed_seconds).toBeGreaterThanOrEqual(0);
  });

  it('passes through modules without quizzes (concept-only lessons)', () => {
    const conceptModules: MicroModuleData[] = [
      { id: 'mm-c', title: 'Pure Concept', content_text: 'No quiz here.' },
    ];
    render(<MicroModuleViewer modules={conceptModules} onComplete={jest.fn()} />);
    expect(screen.getByText('Pure Concept')).toBeInTheDocument();
    expect(screen.queryByText('Knowledge Check')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Complete/ })).toBeEnabled();
  });

  it('renders nothing for an empty module list', () => {
    const { container } = render(<MicroModuleViewer modules={[]} onComplete={jest.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('completing the final module fires onComplete with the payload', async () => {
    const user = userEvent.setup();
    const onComplete = jest.fn();
    render(<MicroModuleViewer modules={[quizModules[0]]} onComplete={onComplete} />);

    await user.click(screen.getByRole('button', { name: 'Two' }));
    await user.click(screen.getByRole('button', { name: /Complete/ }));

    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(onComplete.mock.calls[0][0].total_count).toBe(1);
    expect(onComplete.mock.calls[0][0].correct_count).toBe(1);
  });
});