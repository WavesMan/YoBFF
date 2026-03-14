import { render, screen, fireEvent } from '@testing-library/react';
import { Input } from '../Input';
import { describe, it, expect, vi } from 'vitest';

describe('Input', () => {
  it('renders with label', () => {
    render(<Input label="Username" name="username" />);
    expect(screen.getByLabelText('Username')).toBeInTheDocument();
  });

  it('handles user input', () => {
    const handleChange = vi.fn();
    render(<Input onChange={handleChange} placeholder="Enter text" />);
    const input = screen.getByPlaceholderText('Enter text');
    fireEvent.change(input, { target: { value: 'Hello' } });
    expect(handleChange).toHaveBeenCalledTimes(1);
    expect(input).toHaveValue('Hello');
  });

  it('shows error message', () => {
    render(<Input error="Invalid input" />);
    expect(screen.getByText('Invalid input')).toBeInTheDocument();
    expect(screen.getByRole('textbox')).toHaveClass('has-error');
  });

  it('renders with icon', () => {
    render(<Input icon={<span data-testid="icon">@</span>} />);
    expect(screen.getByTestId('icon')).toBeInTheDocument();
    expect(screen.getByRole('textbox')).toHaveClass('has-icon');
  });

  it('uses provided id', () => {
    render(<Input id="custom-id" label="Label" />);
    expect(screen.getByLabelText('Label')).toHaveAttribute('id', 'custom-id');
  });
});
