import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from './App';

// Mock fetch globally
global.fetch = jest.fn();

const mockFetchResponse = (data: any) => {
  return Promise.resolve({
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data))
  });
};

beforeEach(() => {
  // Clear all mocks before each test
  jest.clearAllMocks();
  (global.fetch as jest.Mock).mockClear();
  
  // Default mock implementation for fetch
  (global.fetch as jest.Mock).mockImplementation(() => 
    mockFetchResponse({ Items: [], Next: undefined, Prev: undefined })
  );
});

afterEach(() => {
  jest.restoreAllMocks();
});

describe('App Component', () => {
  describe('Layout', () => {
    test('renders modern layout with sidebar and main content', () => {
      render(<App />);
      
      // Check for main layout elements
      expect(screen.getByRole('main')).toBeInTheDocument();
      expect(screen.getByRole('complementary')).toBeInTheDocument(); // aside element
    });

    test('renders all demo cards in sidebar', () => {
      render(<App />);
      
      // Check for card titles
      expect(screen.getByText('Hello World Demo')).toBeInTheDocument();
      expect(screen.getByText('Image Generation')).toBeInTheDocument();
      expect(screen.getByText('Scheduled & Async Operations')).toBeInTheDocument();
      expect(screen.getByText('Chat Interface')).toBeInTheDocument();
    });

    test('renders table sections in main content', async () => {
      render(<App />);
      
      // Check for table section titles
      expect(screen.getByText('Workflows')).toBeInTheDocument();
      expect(screen.getByText('States')).toBeInTheDocument();
      
      // Check for table descriptions
      expect(screen.getByText('Manage your workflow definitions')).toBeInTheDocument();
      expect(screen.getByText('View workflow execution states')).toBeInTheDocument();
    });
  });

  describe('Hello World Demo', () => {
    test('renders input with default value', () => {
      render(<App />);
      
      const input = screen.getByPlaceholderText('Enter your name') as HTMLInputElement;
      expect(input).toBeInTheDocument();
      expect(input.value).toBe('Amigo');
    });

    test('updates input value when typing', async () => {
      const user = userEvent.setup();
      render(<App />);
      
      const input = screen.getByPlaceholderText('Enter your name');
      await user.clear(input);
      await user.type(input, 'Test User');
      
      expect(input).toHaveValue('Test User');
    });

    test('renders workflow buttons with proper styling', () => {
      render(<App />);
      
      const runButton = screen.getByText('Run hello world workflow');
      const errorButton = screen.getByText('Run hello world workflow with error');
      
      expect(runButton).toHaveClass('bg-primary'); // Primary button styling
      expect(errorButton).toHaveClass('border'); // Outline variant
    });

    test('shows loading state when running workflow', async () => {
      (global.fetch as jest.Mock)
        .mockImplementationOnce(() => mockFetchResponse('flow-created'))
        .mockImplementationOnce(() => mockFetchResponse({ $type: 'workflow.Done' }));

      render(<App />);
      
      const button = screen.getByText('Run hello world workflow');
      fireEvent.click(button);
      
      expect(screen.getByText('Loading...')).toBeInTheDocument();
      
      await waitFor(() => {
        expect(screen.queryByText('Loading...')).not.toBeInTheDocument();
      });
    });
  });

  describe('Image Generation', () => {
    test('renders width and height inputs', () => {
      render(<App />);
      
      const widthInput = screen.getByPlaceholderText('Width') as HTMLInputElement;
      const heightInput = screen.getByPlaceholderText('Height') as HTMLInputElement;
      
      expect(widthInput).toBeInTheDocument();
      expect(heightInput).toBeInTheDocument();
      expect(widthInput.value).toBe('100');
      expect(heightInput.value).toBe('100');
    });

    test('renders generate image button', () => {
      render(<App />);
      
      const button = screen.getByText('Generate image');
      expect(button).toBeInTheDocument();
      expect(button).toHaveClass('bg-primary'); // Button styling
    });
  });

  describe('Tables', () => {
    test('renders paginated tables with modern styling', async () => {
      render(<App />);
      
      await waitFor(() => {
        // Check for table presence
        const tables = screen.getAllByRole('table');
        expect(tables.length).toBeGreaterThan(0);
        
        // Check for refresh buttons
        const refreshButtons = screen.getAllByText('Refresh');
        expect(refreshButtons.length).toBeGreaterThan(0);
      });
    });

    test('renders action buttons for tables', async () => {
      render(<App />);
      
      await waitFor(() => {
        expect(screen.getByText('Delete selected')).toBeInTheDocument();
        expect(screen.getByText('Delete')).toBeInTheDocument();
        expect(screen.getByText('Try recover')).toBeInTheDocument();
      });
    });
  });

  describe('Modern UI Elements', () => {
    test('uses Lucide icons for visual enhancement', () => {
      render(<App />);
      
      // Check for SVG icons
      const svgElements = document.querySelectorAll('svg');
      expect(svgElements.length).toBeGreaterThan(0);
    });

    test('uses card components for better organization', () => {
      render(<App />);
      
      // Check for card elements with proper styling
      const cards = document.querySelectorAll('.rounded-lg.border.bg-card');
      expect(cards.length).toBeGreaterThan(0);
    });

    test('uses modern Tailwind classes for styling', () => {
      render(<App />);
      
      // Check for modern spacing and layout classes
      expect(document.querySelector('.space-y-4')).toBeInTheDocument();
      expect(document.querySelector('.grid.grid-cols-1')).toBeInTheDocument();
      expect(document.querySelector('.flex.gap-2')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    test('handles backend errors gracefully', async () => {
      // Mock fetch to simulate backend error
      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
      (global.fetch as jest.Mock).mockRejectedValue(new Error('Network error'));

      render(<App />);

      // Wait for the fetch to be called
      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalled();
      });

      // Verify error was logged but app didn't crash
      await waitFor(() => {
        expect(consoleErrorSpy).toHaveBeenCalledWith('Failed to fetch data:', expect.any(Error));
      });

      // Verify the app still renders properly
      expect(screen.getByText('Workflows')).toBeInTheDocument();
      expect(screen.getByText('States')).toBeInTheDocument();

      consoleErrorSpy.mockRestore();
    });
  });
});