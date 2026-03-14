import { useState } from 'react';
import { FiSettings, FiMoreVertical, FiUser, FiTrash2, FiCpu, FiGlobe, FiShield } from 'react-icons/fi';
import { Button } from '../components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '../components/ui/Card';
import { Input } from '../components/ui/Input';
import { Select } from '../components/ui/Select';
import { Switch } from '../components/ui/Switch';
import { Badge } from '../components/ui/Badge';
import { Modal } from '../components/ui/Modal';
import { Drawer } from '../components/ui/Drawer';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/Tabs';
import { useToast } from '../components/ui/Toast';
import { DropdownMenu } from '../components/ui/DropdownMenu';
import { Table } from '../components/ui/Table';

export function UiDemoSection() {
  const toast = useToast();
  const [modalOpen, setModalOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('components');

  const showToast = (type: 'success' | 'error' | 'warning' | 'info') => {
    toast[type](`This is a ${type} toast notification example.`, {
      title: type.charAt(0).toUpperCase() + type.slice(1),
      duration: 3000,
    });
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>UI Component Library</CardTitle>
          <p className="text-sm text-muted-foreground">
            Showcase of all reusable components and layout patterns available in the application.
          </p>
        </CardHeader>
        <CardContent>
          <Tabs
            defaultValue="components"
            value={activeTab}
            onValueChange={setActiveTab}
          >
            <TabsList>
              <TabsTrigger value="components">Atomic Components</TabsTrigger>
              <TabsTrigger value="layouts">Page Layouts</TabsTrigger>
            </TabsList>
            
            <TabsContent value="components">
              <div className="mt-6">
                <ComponentDemo 
                  onOpenModal={() => setModalOpen(true)}
                  onOpenDrawer={() => setDrawerOpen(true)}
                />
              </div>
            </TabsContent>

            <TabsContent value="layouts">
              <div className="mt-6">
                <PageLayoutDemo />
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      <Modal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        title="Demo Modal"
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button variant="primary" onClick={() => { showToast('success'); setModalOpen(false); }}>Confirm</Button>
          </div>
        }
      >
        <div className="p-4">
          <p>This is a demonstration of the Modal component.</p>
          <p className="mt-2 text-sm text-muted-foreground">It supports custom content, headers, and footers.</p>
        </div>
      </Modal>

      <Drawer
        isOpen={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        title="Demo Drawer"
        width={500}
      >
        <div className="p-4 space-y-4">
          <p>This is a side drawer component, useful for detailed configurations or secondary actions without leaving the context.</p>
          <div className="p-4 border border-white/10 rounded bg-white/5">
            <h4 className="font-medium mb-2">Drawer Content</h4>
            <p className="text-sm text-muted-foreground">You can put forms, tables, or any complex content here.</p>
          </div>
          <Button className="w-full" onClick={() => setDrawerOpen(false)}>Close Drawer</Button>
        </div>
      </Drawer>
    </div>
  );
}

function ComponentDemo({ onOpenModal, onOpenDrawer }: { onOpenModal: () => void; onOpenDrawer: () => void }) {
  const toast = useToast();
  const [inputValue, setInputValue] = useState('');
  const [selectValue, setSelectValue] = useState('option1');
  const [switchChecked, setSwitchChecked] = useState(false);

  const showToast = (type: 'success' | 'error' | 'warning' | 'info') => {
    toast[type](`This is a ${type} toast notification example.`, {
      title: type.charAt(0).toUpperCase() + type.slice(1),
      duration: 3000,
    });
  };

  return (
    <div className="space-y-8">
      {/* Buttons */}
      <section>
        <h3 className="text-lg font-medium mb-4">Buttons</h3>
        <div className="flex flex-wrap gap-4 items-center">
          <Button variant="primary">Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="danger">Danger</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="primary" disabled>Disabled</Button>
          <Button variant="primary" size="sm">Small</Button>
          <Button variant="primary" size="lg">Large</Button>
          <Button variant="primary" icon={<FiSettings />}>With Icon</Button>
          <Button variant="primary" loading>Loading</Button>
        </div>
      </section>

      {/* Form Elements */}
      <section>
        <h3 className="text-lg font-medium mb-4">Form Elements</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 items-start">
          <Input 
            label="Text Input" 
            placeholder="Type something..." 
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
          />
          <Input 
            label="Error State" 
            placeholder="Invalid input..." 
            value="Invalid Value"
            error="This field is required"
            readOnly
          />
          <Select
            label="Select Option"
            value={selectValue}
            onChange={(e) => setSelectValue(e.target.value)}
            options={[
              { label: 'Option 1', value: 'option1' },
              { label: 'Option 2', value: 'option2' },
              { label: 'Disabled Option', value: 'option3', disabled: true },
            ]}
          />
          <Select
            label="Error State"
            value=""
            onChange={() => {}}
            error="Please select an option"
            options={[
              { label: 'Select...', value: '' },
            ]}
          />
          <div className="flex items-center gap-4 pt-8">
            <span className="text-sm font-medium">Switch Toggle:</span>
            <Switch 
              checked={switchChecked} 
              onCheckedChange={setSwitchChecked} 
            />
          </div>
        </div>
      </section>

      {/* Badges */}
      <section>
        <h3 className="text-lg font-medium mb-4">Badges</h3>
        <div className="flex gap-4">
          <Badge variant="default">Default</Badge>
          <Badge variant="primary">Primary</Badge>
          <Badge variant="success">Success</Badge>
          <Badge variant="warning">Warning</Badge>
          <Badge variant="error">Error</Badge>
          <Badge variant="primary" showDot>With Dot</Badge>
        </div>
      </section>

      {/* Feedback & Overlays */}
      <section>
        <h3 className="text-lg font-medium mb-4">Feedback & Overlays</h3>
        <div className="flex flex-wrap gap-4">
          <Button onClick={() => showToast('success')}>Success Toast</Button>
          <Button onClick={() => showToast('error')} variant="danger">Error Toast</Button>
          <Button onClick={() => showToast('warning')} variant="secondary">Warning Toast</Button>
          <Button onClick={() => showToast('info')} variant="ghost">Info Toast</Button>
          <Button onClick={onOpenModal} variant="primary">Open Modal</Button>
          <Button onClick={onOpenDrawer} variant="secondary">Open Drawer</Button>
        </div>
      </section>

      {/* Table */}
      <section>
        <h3 className="text-lg font-medium mb-4">Table</h3>
        <div className="space-y-6">
          <Table
            columns={[
              { title: 'Name', key: 'name' },
              { title: 'Status', key: 'status' },
              { title: 'Role', key: 'role' },
            ]}
            data={[
              { id: '1', name: 'John Doe', status: 'Active', role: 'Admin' },
              { id: '2', name: 'Jane Smith', status: 'Inactive', role: 'User' },
            ]}
          />
          <Table
            loading
            columns={[
              { title: 'Name', key: 'name' },
              { title: 'Status', key: 'status' },
              { title: 'Role', key: 'role' },
            ]}
            data={[]}
          />
        </div>
      </section>

      {/* Dropdown */}
      <section>
        <h3 className="text-lg font-medium mb-4">Dropdown Menu</h3>
        <div className="flex gap-4">
          <DropdownMenu 
            trigger={<Button variant="secondary" icon={<FiMoreVertical />}>Actions</Button>}
            items={[
              { label: 'Edit Profile', onClick: () => showToast('info'), icon: <FiUser /> },
              { label: 'Settings', onClick: () => showToast('info'), icon: <FiSettings /> },
              { label: 'Delete Account', onClick: () => showToast('error'), icon: <FiTrash2 />, danger: true },
            ]}
          />
        </div>
      </section>
    </div>
  );
}

function PageLayoutDemo() {
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex justify-between items-center">
            <div>
              <CardTitle>User Details</CardTitle>
              <p className="text-sm text-muted-foreground mt-1">Example of a secondary detail page layout</p>
            </div>
            <div className="flex gap-2">
              <Button variant="secondary" size="sm">Edit</Button>
              <Button variant="primary" size="sm">Save</Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="overview">
            <TabsList>
              <TabsTrigger value="overview">Overview</TabsTrigger>
              <TabsTrigger value="activity">Activity Log</TabsTrigger>
              <TabsTrigger value="settings">Settings</TabsTrigger>
            </TabsList>
            
            <TabsContent value="overview">
              <div className="space-y-6 mt-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-4">
                    <div className="grid grid-cols-3 gap-4 border-b border-white/10 pb-4">
                      <span className="text-muted-foreground text-sm">Full Name</span>
                      <span className="col-span-2 text-sm font-medium">Alice Anderson</span>
                    </div>
                    <div className="grid grid-cols-3 gap-4 border-b border-white/10 pb-4">
                      <span className="text-muted-foreground text-sm">Email</span>
                      <span className="col-span-2 text-sm font-medium">alice@example.com</span>
                    </div>
                    <div className="grid grid-cols-3 gap-4 border-b border-white/10 pb-4">
                      <span className="text-muted-foreground text-sm">Role</span>
                      <span className="col-span-2"><Badge variant="primary">Administrator</Badge></span>
                    </div>
                  </div>
                  <div className="space-y-4">
                    <div className="grid grid-cols-3 gap-4 border-b border-white/10 pb-4">
                      <span className="text-muted-foreground text-sm">Status</span>
                      <span className="col-span-2 flex items-center gap-2">
                        <Badge variant="success" showDot>Active</Badge>
                      </span>
                    </div>
                    <div className="grid grid-cols-3 gap-4 border-b border-white/10 pb-4">
                      <span className="text-muted-foreground text-sm">Last Login</span>
                      <span className="col-span-2 text-sm font-medium">2 hours ago</span>
                    </div>
                  </div>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="activity">
              <div className="mt-6">
                <Table
                  columns={[
                    { title: 'Time', key: 'time' },
                    { title: 'Action', key: 'action' },
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    { title: 'Status', key: 'status', render: (record: any) => record.status },
                  ]}
                  data={[
                    { id: '1', time: '10:00 AM', action: 'Login', status: <Badge variant="success">Success</Badge> },
                    { id: '2', time: '10:05 AM', action: 'Update Profile', status: <Badge variant="success">Success</Badge> },
                    { id: '3', time: '11:30 AM', action: 'Delete Item', status: <Badge variant="error">Failed</Badge> },
                  ]}
                />
              </div>
            </TabsContent>

            <TabsContent value="settings">
              <div className="mt-6 space-y-4 max-w-md">
                <div className="flex items-center justify-between p-4 border border-white/10 rounded-lg">
                  <div>
                    <h4 className="font-medium text-sm">Email Notifications</h4>
                    <p className="text-xs text-muted-foreground">Receive daily summaries</p>
                  </div>
                  <Switch checked={true} />
                </div>
                <div className="flex items-center justify-between p-4 border border-white/10 rounded-lg">
                  <div>
                    <h4 className="font-medium text-sm">2FA Authentication</h4>
                    <p className="text-xs text-muted-foreground">Enhance account security</p>
                  </div>
                  <Switch checked={false} />
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      {/* Server Status Demo */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <FiCpu /> CPU Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">45%</div>
            <p className="text-xs text-muted-foreground mt-1">+2% from last hour</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <FiGlobe /> Network Traffic
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">1.2 GB/s</div>
            <p className="text-xs text-muted-foreground mt-1">Stable</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <FiShield /> Security Threats
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">0</div>
            <p className="text-xs text-success mt-1">System secure</p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
