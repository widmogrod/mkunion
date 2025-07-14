import { parseNaturalLanguage } from './SchedulePopover'

describe('parseNaturalLanguage', () => {
  it('should parse basic patterns', () => {
    expect(parseNaturalLanguage('every minute')).toEqual({
      cron: '* * * * * *',
      description: 'Runs every minute'
    })
    
    expect(parseNaturalLanguage('every 5 minutes')).toEqual({
      cron: '0 */5 * * * *',
      description: 'Runs every 5 minutes'
    })
    
    expect(parseNaturalLanguage('hourly')).toEqual({
      cron: '0 0 * * * *',
      description: 'Runs at the start of every hour'
    })
    
    expect(parseNaturalLanguage('daily')).toEqual({
      cron: '0 0 0 * * *',
      description: 'Runs daily at midnight'
    })
  })
  
  it('should parse time-based patterns', () => {
    expect(parseNaturalLanguage('every 30 seconds')).toEqual({
      cron: '*/30 * * * * *',
      description: 'Runs every 30 seconds'
    })
    
    expect(parseNaturalLanguage('every 2 hours')).toEqual({
      cron: '0 0 */2 * * *',
      description: 'Runs every 2 hours'
    })
  })
  
  it('should parse daily at specific times', () => {
    expect(parseNaturalLanguage('daily at 9am')).toEqual({
      cron: '0 0 9 * * *',
      description: 'Runs daily at 09:00'
    })
    
    expect(parseNaturalLanguage('daily at 2:30pm')).toEqual({
      cron: '0 30 14 * * *',
      description: 'Runs daily at 14:30'
    })
    
    expect(parseNaturalLanguage('daily at 12pm')).toEqual({
      cron: '0 0 12 * * *',
      description: 'Runs daily at 12:00'
    })
    
    expect(parseNaturalLanguage('daily at 12am')).toEqual({
      cron: '0 0 0 * * *',
      description: 'Runs daily at 00:00'
    })
  })
  
  it('should parse weekday patterns', () => {
    expect(parseNaturalLanguage('every monday at 9am')).toEqual({
      cron: '0 0 9 * * 1',
      description: 'Runs every Monday at 09:00'
    })
    
    expect(parseNaturalLanguage('friday at 5:30pm')).toEqual({
      cron: '0 30 17 * * 5',
      description: 'Runs every Friday at 17:30'
    })
    
    expect(parseNaturalLanguage('sundays at 10am')).toEqual({
      cron: '0 0 10 * * 0',
      description: 'Runs every Sunday at 10:00'
    })
  })
  
  it('should handle case insensitive input', () => {
    expect(parseNaturalLanguage('EVERY MINUTE')).toEqual({
      cron: '* * * * * *',
      description: 'Runs every minute'
    })
    
    expect(parseNaturalLanguage('Daily At 9AM')).toEqual({
      cron: '0 0 9 * * *',
      description: 'Runs daily at 09:00'
    })
  })
  
  it('should return null for invalid patterns', () => {
    expect(parseNaturalLanguage('invalid pattern')).toBeNull()
    expect(parseNaturalLanguage('every day at sunset')).toBeNull()
    expect(parseNaturalLanguage('')).toBeNull()
  })
})