import { render, screen } from '@testing-library/react';
import { Badge } from '../Badge';
import { describe, it, expect } from 'vitest';

describe('Badge', () => {
  it('renders content correctly', () => {
    render(<Badge>Status</Badge>);
    expect(screen.getByText('Status')).toBeInTheDocument();
  });

  it('applies variant classes', () => {
    const { container } = render(<Badge variant="success">Success</Badge>);
    expect(container.firstChild).toHaveClass('badge badge-success');
  });

  it('renders dot when showDot is true', () => {
    const { container } = render(<Badge showDot>Online</Badge>);
    expect(container.querySelector('.badge-dot')).toBeInTheDocument();
  });

  it('does not render dot by default', () => {
    const { container } = render(<Badge>Offline</Badge>);
    expect(container.querySelector('.badge-dot')).not.toBeInTheDocument();
  });
});
