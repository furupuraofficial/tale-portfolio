import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, expect, test, vi } from 'vitest';
import App from './App';

const rules = [
  {
    id: 'bath-rule',
    keywords: ['温泉', '入浴'],
    question: '温泉はどこですか？',
    answer: '温泉は2階にあります。',
    arActions: [{ type: 'POINT', target: 'stairs', params: { floor: '2' } }],
  },
];

beforeEach(() => {
  global.fetch = vi.fn().mockResolvedValue({
    ok: true,
    json: async () => rules,
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

test('loads rules and selects one for editing', async () => {
  const user = userEvent.setup();
  render(<App />);

  const question = await screen.findByText('温泉はどこですか？');
  expect(global.fetch).toHaveBeenCalledWith('/rule/items');

  await user.click(question);

  const form = screen.getByRole('form');
  expect(within(form).getByDisplayValue('bath-rule')).toBeInTheDocument();
  expect(within(form).getByDisplayValue('温泉, 入浴')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'ルールを更新' })).toBeEnabled();
  expect(global.fetch).toHaveBeenCalledTimes(1);
});

test('shows an API error without hiding the editor', async () => {
  global.fetch.mockResolvedValueOnce({ ok: false });

  render(<App />);

  expect(await screen.findByText('ルールの読み込みに失敗しました')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'ルールを作成' })).toBeEnabled();
});
