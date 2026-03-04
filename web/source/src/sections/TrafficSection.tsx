import { useState } from 'react'
import { SiteList } from './traffic/SiteList'
import { SiteConfigDrawer } from './traffic/SiteConfigDrawer'

type TrafficSectionProps = {
  token: string
  operator: string
}

export function TrafficSection({ token, operator }: TrafficSectionProps) {
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null)

  return (
    <div className="traffic-section">
      <SiteList 
        token={token} 
        onSelectSite={setSelectedSiteId} 
      />
      
      {selectedSiteId && (
        <SiteConfigDrawer
          token={token}
          operator={operator}
          siteId={selectedSiteId}
          onClose={() => setSelectedSiteId(null)}
        />
      )}
    </div>
  )
}
